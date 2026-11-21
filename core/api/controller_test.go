package api

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/arafsk/munna-proxy/core/socks5"
)

func startEcho(t *testing.T) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
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
	return listener.Addr().String()
}

func TestHostAndClientRoundTrip(t *testing.T) {
	controller := New()
	defer controller.Close()
	echo := startEcho(t)

	host, err := controller.StartHost(HostOptions{
		Listen:      "127.0.0.1:0",
		User:        "alice",
		Password:    "correct-horse",
		DisableMDNS: true,
	})
	if err != nil {
		t.Fatalf("StartHost: %v", err)
	}
	if !host.Running || host.Password != "correct-horse" || !strings.HasPrefix(host.Invite, "munna://") {
		t.Fatalf("host info = %+v", host)
	}
	if !strings.HasPrefix(host.QR, "data:image/png;base64,") {
		t.Fatalf("QR is not a PNG data URL")
	}

	client, err := controller.StartClient(ClientOptions{
		Host:     host.Listen,
		User:     "alice",
		Password: "correct-horse",
		Listen:   "127.0.0.1:0",
	})
	if err != nil {
		t.Fatalf("StartClient: %v", err)
	}
	if !client.Running {
		t.Fatalf("client not running: %+v", client)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	upstream := socks5.Upstream{Address: client.Listen}
	conn, err := upstream.Connect(ctx, echo)
	if err != nil {
		t.Fatalf("Connect through client forwarder: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Write([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 2)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(buffer) != "hi" {
		t.Fatalf("echo = %q", buffer)
	}

	for _, entry := range controller.Logs(0) {
		if strings.Contains(entry.Message, "correct-horse") {
			t.Fatalf("password leaked into logs: %q", entry.Message)
		}
	}
}

func TestClientRejectsWrongPassword(t *testing.T) {
	controller := New()
	defer controller.Close()

	host, err := controller.StartHost(HostOptions{Listen: "127.0.0.1:0", User: "alice", Password: "right", DisableMDNS: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.StartClient(ClientOptions{Host: host.Listen, User: "alice", Password: "wrong", Listen: "127.0.0.1:0"})
	if err == nil {
		t.Fatal("StartClient with wrong password succeeded")
	}
	if controller.ClientStatus().Running {
		t.Fatal("client reported running after failed start")
	}
}

func TestHostGeneratesPasswordWhenEmpty(t *testing.T) {
	controller := New()
	defer controller.Close()

	host, err := controller.StartHost(HostOptions{Listen: "127.0.0.1:0", DisableMDNS: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(host.Password) < 20 || host.User != "munna" {
		t.Fatalf("generated credentials = %q / %q", host.User, host.Password)
	}
}

func TestNoAuthRequiresLoopback(t *testing.T) {
	controller := New()
	defer controller.Close()

	_, err := controller.StartHost(HostOptions{Listen: "0.0.0.0:0", NoAuth: true, DisableMDNS: true})
	if err == nil {
		t.Fatal("no-auth on all interfaces was allowed")
	}
}

func TestClientListenMustBeLoopback(t *testing.T) {
	controller := New()
	defer controller.Close()

	host, err := controller.StartHost(HostOptions{Listen: "127.0.0.1:0", NoAuth: true, DisableMDNS: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = controller.StartClient(ClientOptions{Host: host.Listen, Listen: "0.0.0.0:0"})
	if err == nil {
		t.Fatal("client forwarder bound to a non-loopback address")
	}
}

func TestSingleSessionPerKind(t *testing.T) {
	controller := New()
	defer controller.Close()

	if _, err := controller.StartHost(HostOptions{Listen: "127.0.0.1:0", NoAuth: true, DisableMDNS: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.StartHost(HostOptions{Listen: "127.0.0.1:0", NoAuth: true, DisableMDNS: true}); err == nil {
		t.Fatal("second host started")
	}
	controller.StopHost()
	if controller.HostStatus().Running {
		t.Fatal("host still running after StopHost")
	}
}

func TestResolveHostAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "host:port", input: "192.168.1.10:1080", want: "192.168.1.10:1080"},
		{name: "invite", input: "munna://abc@192.168.1.10:1080?v=1", want: "192.168.1.10:1080"},
		{name: "invite without lan address", input: "munna://abc?v=1", wantErr: true},
		{name: "empty", input: "  ", wantErr: true},
		{name: "no port", input: "192.168.1.10", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveHostAddress(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("ResolveHostAddress(%q) = %q, want error", test.input, got)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("ResolveHostAddress(%q) = %q, %v; want %q", test.input, got, err, test.want)
			}
		})
	}
}

func TestIsLoopbackHost(t *testing.T) {
	for host, want := range map[string]bool{
		"127.0.0.1": true,
		"::1":       true,
		"localhost": true,
		"0.0.0.0":   false,
		"":          false,
		"10.0.0.1":  false,
	} {
		if got := IsLoopbackHost(host); got != want {
			t.Errorf("IsLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestLogBufferKeepsCompleteLines(t *testing.T) {
	var buffer logBuffer
	_, _ = buffer.Write([]byte("first\nsec"))
	_, _ = buffer.Write([]byte("ond\n"))
	entries := buffer.since(0)
	if len(entries) != 2 || entries[0].Message != "first" || entries[1].Message != "second" {
		t.Fatalf("entries = %+v", entries)
	}
	if got := buffer.since(entries[0].Seq); len(got) != 1 || got[0].Message != "second" {
		t.Fatalf("since = %+v", got)
	}
}
