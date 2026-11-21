package socks5

import (
	"io"
	"net"
	"sync"
	"time"
)

// relay copies bytes in both directions until both sides finish or the
// connection goes idle.
func relay(client, upstream net.Conn, idleTimeout time.Duration) {
	clientConn := &idleConn{Conn: client, timeout: idleTimeout}
	upstreamConn := &idleConn{Conn: upstream, timeout: idleTimeout}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		copyAndCloseWrite(upstreamConn, clientConn)
	}()
	go func() {
		defer wg.Done()
		copyAndCloseWrite(clientConn, upstreamConn)
	}()
	wg.Wait()
}

func copyAndCloseWrite(dst, src net.Conn) {
	_, _ = io.Copy(dst, src)
	if closer, ok := dst.(interface{ CloseWrite() error }); ok {
		_ = closer.CloseWrite()
		return
	}
	_ = dst.Close()
}

// idleConn refreshes a deadline before every read and write so that a
// connection with no traffic is eventually closed.
type idleConn struct {
	net.Conn
	timeout time.Duration
}

func (c *idleConn) Read(p []byte) (int, error) {
	_ = c.SetReadDeadline(time.Now().Add(c.timeout))
	return c.Conn.Read(p)
}

func (c *idleConn) Write(p []byte) (int, error) {
	_ = c.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.Conn.Write(p)
}

func (c *idleConn) CloseWrite() error {
	if closer, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return closer.CloseWrite()
	}
	return c.Close()
}
