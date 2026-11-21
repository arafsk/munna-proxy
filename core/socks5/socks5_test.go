package socks5

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/arafsk/munna-proxy/core/dialer"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
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

func startServer(t *testing.T, cfg Config) (address string, stop func()) {
	t.Helper()
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:0"
	}
	if cfg.Dialer == nil {
		cfg.Dialer = dialer.New(dialer.Config{})
	}
	if cfg.Logger == nil {
		cfg.Logger = testLogger()
	}
	server, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	listener, err := server.Listen()
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := server.Serve(ctx, listener); err != nil {
			t.Errorf("Serve: %v", err)
		}
	}()
	return listener.Addr().String(), func() {
		cancel()
		<-done
	}
}

func openProxy(t *testing.T, address, username, password string) net.Conn {
	t.Helper()
	netDialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := netDialer.DialContext(context.Background(), "tcp", address)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	method := byte(methodNoAuth)
	greeting := []byte{version5, 1, methodNoAuth}
	if username != "" {
		method = methodUserPass
		greeting = []byte{version5, 1, methodUserPass}
	}
	if _, err := conn.Write(greeting); err != nil {
		t.Fatalf("write greeting: %v", err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("read method: %v", err)
	}
	if reply[0] != version5 || reply[1] != method {
		t.Fatalf("method reply = %v, want [5 %d]", reply, method)
	}

	if method == methodUserPass {
		auth := []byte{authVersion, byte(len(username))}
		auth = append(auth, username...)
		auth = append(auth, byte(len(password)))
		auth = append(auth, password...)
		if _, err := conn.Write(auth); err != nil {
			t.Fatalf("write auth: %v", err)
		}
		if _, err := io.ReadFull(conn, reply); err != nil {
			t.Fatalf("read auth status: %v", err)
		}
		if reply[1] != 0x00 {
			t.Fatalf("auth status = %#x, want success", reply[1])
		}
	}
	return conn
}

func requestConnect(t *testing.T, conn net.Conn, target string) byte {
	t.Helper()
	host, portString, err := net.SplitHostPort(target)
	if err != nil {
		t.Fatalf("split target: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	request := []byte{version5, cmdConnect, 0x00}
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			request = append(request, atypIPv4)
			request = append(request, ip4...)
		} else {
			request = append(request, atypIPv6)
			request = append(request, ip.To16()...)
		}
	} else {
		request = append(request, atypDomain, byte(len(host)))
		request = append(request, host...)
	}
	request = append(request, byte(port>>8), byte(port))

	if _, err := conn.Write(request); err != nil {
		t.Fatalf("write request: %v", err)
	}
	return readReply(t, conn)
}

func readReply(t *testing.T, conn net.Conn) byte {
	t.Helper()
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		t.Fatalf("read reply header: %v", err)
	}
	if header[0] != version5 {
		t.Fatalf("reply version = %#x, want 5", header[0])
	}
	var addressLength int
	switch header[3] {
	case atypIPv4:
		addressLength = 4
	case atypIPv6:
		addressLength = 16
	case atypDomain:
		var length [1]byte
		if _, err := io.ReadFull(conn, length[:]); err != nil {
			t.Fatalf("read reply domain length: %v", err)
		}
		addressLength = int(length[0])
	default:
		t.Fatalf("reply atyp = %#x", header[3])
	}
	if _, err := io.ReadFull(conn, make([]byte, addressLength+2)); err != nil {
		t.Fatalf("read reply address: %v", err)
	}
	return header[1]
}

func TestConnectNoAuth(t *testing.T) {
	echo := startEcho(t)
	proxy, stop := startServer(t, Config{})
	defer stop()

	conn := openProxy(t, proxy, "", "")
	if reply := requestConnect(t, conn, echo.Addr().String()); reply != repSuccess {
		t.Fatalf("reply = %#x, want success", reply)
	}
	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buffer := make([]byte, 5)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buffer) != "hello" {
		t.Fatalf("echo = %q, want hello", buffer)
	}
}

func TestConnectWithAuth(t *testing.T) {
	auth, err := NewStaticAuth("user", "secret")
	if err != nil {
		t.Fatal(err)
	}
	echo := startEcho(t)
	proxy, stop := startServer(t, Config{Authenticator: auth})
	defer stop()

	conn := openProxy(t, proxy, "user", "secret")
	if reply := requestConnect(t, conn, echo.Addr().String()); reply != repSuccess {
		t.Fatalf("reply = %#x, want success", reply)
	}
}

// attemptAuth performs the greeting and RFC 1929 exchange, returning the
// authentication status byte (0x00 success, 0x01 failure).
func attemptAuth(t *testing.T, proxy, username, password string) byte {
	t.Helper()
	netDialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := netDialer.DialContext(context.Background(), "tcp", proxy)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write([]byte{version5, 1, methodUserPass}); err != nil {
		t.Fatalf("write greeting: %v", err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("read method: %v", err)
	}
	if reply[1] != methodUserPass {
		t.Fatalf("method = %#x, want user/pass", reply[1])
	}

	auth := []byte{authVersion, byte(len(username))}
	auth = append(auth, username...)
	auth = append(auth, byte(len(password)))
	auth = append(auth, password...)
	if _, err := conn.Write(auth); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("read auth status: %v", err)
	}
	return reply[1]
}

func TestAuthFailure(t *testing.T) {
	auth, err := NewStaticAuth("user", "secret")
	if err != nil {
		t.Fatal(err)
	}
	proxy, stop := startServer(t, Config{Authenticator: auth})
	defer stop()

	netDialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := netDialer.DialContext(context.Background(), "tcp", proxy)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write([]byte{version5, 1, methodUserPass}); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Write([]byte{authVersion, 4, 'u', 's', 'e', 'r', 5, 'w', 'r', 'o', 'n', 'g'}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	if reply[1] != 0x01 {
		t.Fatalf("auth status = %#x, want failure", reply[1])
	}
}

func TestAuthLockout(t *testing.T) {
	auth, err := NewStaticAuth("user", "secret")
	if err != nil {
		t.Fatal(err)
	}
	limiter := NewAuthLimiter(2, time.Minute, time.Minute)
	proxy, stop := startServer(t, Config{Authenticator: auth, AuthLimiter: limiter})
	defer stop()

	for i := 0; i < 2; i++ {
		if status := attemptAuth(t, proxy, "user", "wrong"); status != 0x01 {
			t.Fatalf("attempt %d status = %#x, want failure", i+1, status)
		}
	}
	if status := attemptAuth(t, proxy, "user", "secret"); status != 0x01 {
		t.Fatalf("locked-out status = %#x, want failure", status)
	}
}

func TestNoAcceptableMethod(t *testing.T) {
	auth, err := NewStaticAuth("user", "secret")
	if err != nil {
		t.Fatal(err)
	}
	proxy, stop := startServer(t, Config{Authenticator: auth})
	defer stop()

	netDialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := netDialer.DialContext(context.Background(), "tcp", proxy)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write([]byte{version5, 1, methodNoAuth}); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 2)
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	if reply[1] != methodNoAcceptable {
		t.Fatalf("method = %#x, want %#x", reply[1], methodNoAcceptable)
	}
}

func TestUnsupportedCommand(t *testing.T) {
	echo := startEcho(t)
	proxy, stop := startServer(t, Config{})
	defer stop()

	conn := openProxy(t, proxy, "", "")
	host, portString, err := net.SplitHostPort(echo.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	ip := net.ParseIP(host).To4()
	port, _ := strconv.Atoi(portString)
	request := []byte{version5, cmdBind, 0x00, atypIPv4}
	request = append(request, ip...)
	request = append(request, byte(port>>8), byte(port))
	if _, err := conn.Write(request); err != nil {
		t.Fatal(err)
	}
	if reply := readReply(t, conn); reply != repCommandNotSupported {
		t.Fatalf("reply = %#x, want command not supported", reply)
	}
}

func TestConnectDomain(t *testing.T) {
	echo := startEcho(t)
	_, port, err := net.SplitHostPort(echo.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	proxy, stop := startServer(t, Config{})
	defer stop()

	conn := openProxy(t, proxy, "", "")
	if reply := requestConnect(t, conn, net.JoinHostPort("localhost", port)); reply != repSuccess {
		t.Fatalf("reply = %#x, want success", reply)
	}
}

func TestConnectIPv6(t *testing.T) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
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

	proxy, stop := startServer(t, Config{})
	defer stop()

	conn := openProxy(t, proxy, "", "")
	if reply := requestConnect(t, conn, listener.Addr().String()); reply != repSuccess {
		t.Fatalf("reply = %#x, want success", reply)
	}
}

type errorDialer struct {
	err error
}

func (d errorDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, d.err
}

func TestPolicyDeniedReply(t *testing.T) {
	proxy, stop := startServer(t, Config{Dialer: errorDialer{err: dialer.ErrPolicyDenied}})
	defer stop()

	conn := openProxy(t, proxy, "", "")
	if reply := requestConnect(t, conn, "127.0.0.1:80"); reply != repNotAllowed {
		t.Fatalf("reply = %#x, want not allowed", reply)
	}
}

func TestReplyForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want byte
	}{
		{"policy", dialer.ErrPolicyDenied, repNotAllowed},
		{"dns", dialer.ErrDNS, repHostUnreachable},
		{"deadline", context.DeadlineExceeded, repTTLExpired},
		{"refused", syscall.ECONNREFUSED, repConnectionRefused},
		{"network", syscall.ENETUNREACH, repNetworkUnreachable},
		{"host", syscall.EHOSTUNREACH, repHostUnreachable},
		{"other", errors.New("boom"), repGeneralFailure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := replyForError(test.err); got != test.want {
				t.Fatalf("replyForError(%v) = %#x, want %#x", test.err, got, test.want)
			}
		})
	}
}

func FuzzReadGreeting(f *testing.F) {
	f.Add([]byte{version5, 1, methodNoAuth})
	f.Add([]byte{version5, 0})
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = readGreeting(bytes.NewReader(data))
	})
}

func FuzzReadAuth(f *testing.F) {
	f.Add([]byte{authVersion, 1, 'u', 1, 'p'})
	f.Add([]byte{})
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _, _ = readAuth(bytes.NewReader(data))
	})
}

func FuzzReadRequest(f *testing.F) {
	f.Add([]byte{version5, cmdConnect, 0x00, atypIPv4, 127, 0, 0, 1, 0, 80})
	f.Add([]byte{version5, cmdConnect, 0x00, atypDomain, 11, 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 'c', 'o', 'm', 0, 80})
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = readRequest(bytes.NewReader(data))
	})
}
