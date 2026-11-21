package socks5

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestUpstreamConnectNoAuth(t *testing.T) {
	echo := startEcho(t)
	proxy, stop := startServer(t, Config{})
	defer stop()

	conn, err := Upstream{Address: proxy}.Connect(context.Background(), echo.Addr().String())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer func() { _ = conn.Close() }()
	assertEcho(t, conn)
}

func TestUpstreamConnectWithAuthAndDomain(t *testing.T) {
	auth, err := NewStaticAuth("alice", "pw")
	if err != nil {
		t.Fatal(err)
	}
	echo := startEcho(t)
	_, port, err := net.SplitHostPort(echo.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	proxy, stop := startServer(t, Config{Authenticator: auth})
	defer stop()

	upstream := Upstream{Address: proxy, Credentials: &Credentials{Username: "alice", Password: "pw"}}
	conn, err := upstream.Connect(context.Background(), net.JoinHostPort("localhost", port))
	if err != nil {
		t.Fatalf("Connect via domain: %v", err)
	}
	defer func() { _ = conn.Close() }()
	assertEcho(t, conn)
}

func TestUpstreamRejectsWrongPassword(t *testing.T) {
	auth, err := NewStaticAuth("alice", "pw")
	if err != nil {
		t.Fatal(err)
	}
	proxy, stop := startServer(t, Config{Authenticator: auth})
	defer stop()

	upstream := Upstream{Address: proxy, Credentials: &Credentials{Username: "alice", Password: "bad"}}
	_, err = upstream.Connect(context.Background(), "127.0.0.1:9")
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("Connect with wrong password = %v, want ErrRejected", err)
	}
}

func TestUpstreamReportsPolicyDenial(t *testing.T) {
	proxy, stop := startServer(t, Config{})
	defer stop()

	_, err := Upstream{Address: proxy}.Connect(context.Background(), "169.254.169.254:80")
	var reply ReplyError
	if !errors.As(err, &reply) || reply.Code != repNotAllowed {
		t.Fatalf("Connect to metadata = %v, want reply %#x", err, repNotAllowed)
	}
}

func TestChainForwardsThroughHost(t *testing.T) {
	echo := startEcho(t)
	auth, err := NewStaticAuth("alice", "pw")
	if err != nil {
		t.Fatal(err)
	}
	host, stopHost := startServer(t, Config{Authenticator: auth})
	defer stopHost()

	chain := Chain{Upstream: Upstream{Address: host, Credentials: &Credentials{Username: "alice", Password: "pw"}}}
	local, stopLocal := startServerWithDialer(t, Config{}, chain)
	defer stopLocal()

	conn := openProxy(t, local, "", "")
	if reply := requestConnect(t, conn, echo.Addr().String()); reply != repSuccess {
		t.Fatalf("reply through chain = %#x, want success", reply)
	}
	assertEchoOn(t, conn)
}

func TestChainRejectsUnsupportedNetwork(t *testing.T) {
	chain := Chain{Upstream: Upstream{Address: "127.0.0.1:1"}}
	_, err := chain.DialContext(context.Background(), "udp", "127.0.0.1:53")
	if err == nil {
		t.Fatal("expected error for udp")
	}
}

func TestUpstreamTimesOutWithoutServer(t *testing.T) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := (Upstream{Address: address}).Connect(ctx, "127.0.0.1:80"); err == nil {
		t.Fatal("expected connect failure")
	}
}

func startServerWithDialer(t *testing.T, cfg Config, d Dialer) (string, func()) {
	t.Helper()
	cfg.Dialer = d
	return startServer(t, cfg)
}

func assertEcho(t *testing.T, conn net.Conn) {
	t.Helper()
	assertEchoOn(t, conn)
}

func assertEchoOn(t *testing.T, conn net.Conn) {
	t.Helper()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buffer) != "ping" {
		t.Fatalf("echo = %q", buffer)
	}
}
