package socks5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"time"
)

// Credentials are the RFC 1929 username and password for an upstream proxy.
type Credentials struct {
	Username string
	Password string
}

// Upstream is a SOCKS5 proxy that a client connects through.
type Upstream struct {
	// Address is host:port of the SOCKS5 server.
	Address string
	// Credentials enables username/password authentication when non-nil.
	Credentials *Credentials
	// Timeout bounds the TCP connect and the handshake. Defaults to 10s.
	Timeout time.Duration
}

// ErrRejected is returned when the upstream refuses the credentials or the
// greeting.
var ErrRejected = errors.New("socks5: upstream rejected authentication")

// ReplyError is a non-success reply from the upstream server.
type ReplyError struct {
	Code byte
}

func (e ReplyError) Error() string {
	return fmt.Sprintf("socks5: upstream reply %#x", e.Code)
}

// Authenticate connects to the upstream and completes the greeting and, when
// configured, RFC 1929 authentication. The returned connection is ready for a
// request.
func (u Upstream) Authenticate(ctx context.Context) (net.Conn, error) {
	timeout := u.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", u.Address)
	if err != nil {
		return nil, fmt.Errorf("socks5: connect upstream: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	if err := u.handshake(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func (u Upstream) handshake(conn net.Conn) error {
	methods := []byte{methodNoAuth}
	if u.Credentials != nil {
		methods = []byte{methodNoAuth, methodUserPass}
	}
	count, err := lengthByte(len(methods))
	if err != nil {
		return err
	}
	greeting := append([]byte{version5, count}, methods...)
	if _, err := conn.Write(greeting); err != nil {
		return fmt.Errorf("socks5: write greeting: %w", err)
	}

	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return fmt.Errorf("socks5: read method: %w", err)
	}
	if reply[0] != version5 {
		return errUnsupportedVersion
	}

	switch reply[1] {
	case methodNoAuth:
		return nil
	case methodUserPass:
		if u.Credentials == nil {
			return ErrRejected
		}
		return u.authenticate(conn)
	default:
		return ErrRejected
	}
}

func (u Upstream) authenticate(conn net.Conn) error {
	user, pass := u.Credentials.Username, u.Credentials.Password
	if len(user) == 0 {
		return fmt.Errorf("%w: empty username", ErrRejected)
	}
	userLength, err := lengthByte(len(user))
	if err != nil {
		return err
	}
	passLength, err := lengthByte(len(pass))
	if err != nil {
		return err
	}
	request := []byte{authVersion, userLength}
	request = append(request, user...)
	request = append(request, passLength)
	request = append(request, pass...)
	if _, err := conn.Write(request); err != nil {
		return fmt.Errorf("socks5: write auth: %w", err)
	}
	status := make([]byte, 2)
	if _, err := io.ReadFull(conn, status); err != nil {
		return fmt.Errorf("socks5: read auth status: %w", err)
	}
	if status[1] != 0x00 {
		return ErrRejected
	}
	return nil
}

// Connect opens a tunnel through the upstream to target (host:port). Names
// are sent to the upstream unresolved so the host performs DNS.
func (u Upstream) Connect(ctx context.Context, target string) (net.Conn, error) {
	host, portString, err := net.SplitHostPort(target)
	if err != nil {
		return nil, fmt.Errorf("socks5: invalid target %q: %w", target, err)
	}
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("socks5: invalid port in %q: %w", target, err)
	}

	request := []byte{version5, cmdConnect, 0x00}
	if addr, parseErr := netip.ParseAddr(host); parseErr == nil {
		addr = addr.Unmap()
		if addr.Is4() {
			raw := addr.As4()
			request = append(request, atypIPv4)
			request = append(request, raw[:]...)
		} else {
			raw := addr.As16()
			request = append(request, atypIPv6)
			request = append(request, raw[:]...)
		}
	} else {
		hostLength, err := lengthByte(len(host))
		if err != nil || hostLength == 0 {
			return nil, fmt.Errorf("socks5: invalid host name length %d", len(host))
		}
		request = append(request, atypDomain, hostLength)
		request = append(request, host...)
	}
	request = binary.BigEndian.AppendUint16(request, uint16(port))

	conn, err := u.Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write(request); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("socks5: write request: %w", err)
	}
	if err := readConnectReply(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

func readConnectReply(conn net.Conn) error {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return fmt.Errorf("socks5: read reply: %w", err)
	}
	if header[0] != version5 {
		return errUnsupportedVersion
	}
	var addressLength int
	switch header[3] {
	case atypIPv4:
		addressLength = 4
	case atypIPv6:
		addressLength = 16
	case atypDomain:
		length := make([]byte, 1)
		if _, err := io.ReadFull(conn, length); err != nil {
			return fmt.Errorf("socks5: read reply address: %w", err)
		}
		addressLength = int(length[0])
	default:
		return errUnsupportedAddress
	}
	if _, err := io.ReadFull(conn, make([]byte, addressLength+2)); err != nil {
		return fmt.Errorf("socks5: read reply address: %w", err)
	}
	if header[1] != repSuccess {
		return ReplyError{Code: header[1]}
	}
	return nil
}

// lengthByte converts a length to the one-byte form used by SOCKS5 fields.
func lengthByte(n int) (byte, error) {
	if n < 0 || n > 255 {
		return 0, fmt.Errorf("socks5: length %d does not fit in one byte", n)
	}
	return byte(n), nil
}

// Chain adapts an Upstream to the Dialer interface used by Server, so a local
// SOCKS5 server can forward every connection through a remote host.
type Chain struct {
	Upstream Upstream
}

// DialContext implements Dialer. Only TCP is supported.
func (c Chain) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("socks5: unsupported network %q", network)
	}
	return c.Upstream.Connect(ctx, address)
}
