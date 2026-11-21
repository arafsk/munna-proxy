package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestPolicyDeniesMetadata(t *testing.T) {
	policy := DefaultPolicy()
	targets := []string{
		"169.254.169.254",
		"169.254.170.2",
		"100.100.100.200",
		"192.0.0.192",
		"fd00:ec2::254",
	}
	for _, raw := range targets {
		addr := netip.MustParseAddr(raw)
		if err := policy.Check(addr, 80); !errors.Is(err, ErrPolicyDenied) {
			t.Errorf("Check(%s) = %v, want ErrPolicyDenied", raw, err)
		}
	}
}

func TestPolicyAllowsLoopbackAndPublic(t *testing.T) {
	policy := DefaultPolicy()
	targets := []string{
		"127.0.0.1",
		"::1",
		"93.184.216.34",
		"2606:2800:220:1:248:1893:25c8:1946",
	}
	for _, raw := range targets {
		addr := netip.MustParseAddr(raw)
		if err := policy.Check(addr, 443); err != nil {
			t.Errorf("Check(%s) = %v, want nil", raw, err)
		}
	}
}

func TestPolicyDeniesPort(t *testing.T) {
	policy := DefaultPolicy()
	policy.DeniedPorts = map[uint16]struct{}{25: {}}
	err := policy.Check(netip.MustParseAddr("127.0.0.1"), 25)
	if !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("Check(port 25) = %v, want ErrPolicyDenied", err)
	}
}

func TestPolicyDeniesLoopbackWhenDisabled(t *testing.T) {
	policy := DefaultPolicy()
	policy.AllowLoopback = false
	err := policy.Check(netip.MustParseAddr("127.0.0.1"), 80)
	if !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("Check(loopback) = %v, want ErrPolicyDenied", err)
	}
}

func startEcho(t *testing.T) net.Listener {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	return listener
}

func TestDialLoopback(t *testing.T) {
	listener := startEcho(t)
	dialer := New(Config{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := dialer.DialContext(ctx, "tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buffer) != "ping" {
		t.Fatalf("echo = %q, want %q", buffer, "ping")
	}
}

func TestDialDomain(t *testing.T) {
	listener := startEcho(t)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	dialer := New(Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		t.Fatalf("DialContext(localhost): %v", err)
	}
	_ = conn.Close()
}

func TestDialDeniesMetadata(t *testing.T) {
	dialer := New(Config{})
	_, err := dialer.DialContext(context.Background(), "tcp", "169.254.169.254:80")
	if !errors.Is(err, ErrPolicyDenied) {
		t.Fatalf("DialContext(metadata) = %v, want ErrPolicyDenied", err)
	}
}

func TestDialDNSFailure(t *testing.T) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("resolver disabled for test")
		},
	}
	dialer := New(Config{Resolver: resolver})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := dialer.DialContext(ctx, "tcp", "example.invalid:80")
	if !errors.Is(err, ErrDNS) {
		t.Fatalf("DialContext(bad name) = %v, want ErrDNS", err)
	}
}
