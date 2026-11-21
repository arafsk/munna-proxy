package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/arafsk/munna-proxy/core/dialer"
	"github.com/arafsk/munna-proxy/core/discovery"
	"github.com/arafsk/munna-proxy/core/socks5"
)

func runServe(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	listen := flags.String("listen", "0.0.0.0:1080", "address to listen on")
	user := flags.String("user", "munna", "username for SOCKS5 authentication")
	password := flags.String("password", "", "password (generated when empty)")
	noAuth := flags.Bool("no-auth", false, "disable authentication (loopback only)")
	advertise := flags.String("advertise", "", "host:port to advertise in the invite")
	name := flags.String("name", "", "host name shown to clients")
	mdns := flags.Bool("mdns", true, "advertise the host over mDNS")
	debug := flags.Bool("debug", false, "enable debug logging")

	if err := flags.Parse(args); err != nil {
		return err
	}

	logger := newLogger(*debug)

	host, portString, err := net.SplitHostPort(*listen)
	if err != nil {
		return fmt.Errorf("invalid -listen: %w", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid -listen port %q", portString)
	}

	loopbackOnly := isLoopbackHost(host)
	if *noAuth && !loopbackOnly {
		return errors.New("-no-auth is only allowed when listening on a loopback address")
	}

	var authenticator socks5.Authenticator
	effectivePassword := ""
	if !*noAuth {
		effectivePassword = *password
		if effectivePassword == "" {
			effectivePassword, err = randomPassword()
			if err != nil {
				return err
			}
		}
		staticAuth, err := socks5.NewStaticAuth(*user, effectivePassword)
		if err != nil {
			return err
		}
		authenticator = staticAuth
	}

	server, err := socks5.New(socks5.Config{
		ListenAddr:    *listen,
		Authenticator: authenticator,
		Dialer:        dialer.New(dialer.Config{Logger: logger}),
		Logger:        logger,
	})
	if err != nil {
		return err
	}

	listener, err := server.Listen()
	if err != nil {
		return err
	}

	actualPort := port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}
	actualPortString := strconv.Itoa(actualPort)

	hostID, err := discovery.NewHostID()
	if err != nil {
		_ = listener.Close()
		return err
	}
	hostName := *name
	if hostName == "" {
		hostName, err = os.Hostname()
		if err != nil || hostName == "" {
			hostName = "munna"
		}
	}

	advertisedHost := *advertise
	if advertisedHost == "" {
		advertisedHost = net.JoinHostPort(advertiseIP(host), actualPortString)
	} else if _, _, err := net.SplitHostPort(advertisedHost); err != nil {
		_ = listener.Close()
		return fmt.Errorf("invalid -advertise: %w", err)
	}

	inviteURI, err := discovery.Invite{
		Version: 1,
		HostID:  hostID,
		Host:    advertisedHost,
		Name:    hostName,
	}.String()
	if err != nil {
		_ = listener.Close()
		return err
	}

	printBanner(os.Stdout, banner{
		Name:      hostName,
		Listen:    *listen,
		Advertise: advertisedHost,
		User:      *user,
		Password:  effectivePassword,
		NoAuth:    *noAuth,
		Invite:    inviteURI,
	})

	if *mdns {
		advertiser, err := discovery.Advertise(hostName, actualPort, []string{
			"v=1",
			"id=" + hostID,
			"name=" + hostName,
		})
		if err != nil {
			logger.Warn("mDNS advertisement failed", "err", err)
		} else {
			defer advertiser.Close()
			logger.Debug("mDNS advertised", "service", discovery.ServiceType, "instance", hostName)
		}
	}

	logger.Info("host listening", "listen", *listen, "advertise", advertisedHost, "auth", !*noAuth)
	return server.Serve(ctx, listener)
}

type banner struct {
	Name      string
	Listen    string
	Advertise string
	User      string
	Password  string
	NoAuth    bool
	Invite    string
}

func printBanner(w io.Writer, b banner) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\nMunna host %q\n", b.Name)
	fmt.Fprintf(&sb, "  Listen    %s\n", b.Listen)
	fmt.Fprintf(&sb, "  Advertise %s\n", b.Advertise)
	if b.NoAuth {
		sb.WriteString("  Auth      disabled (loopback only)\n")
	} else {
		fmt.Fprintf(&sb, "  User      %s\n", b.User)
		fmt.Fprintf(&sb, "  Password  %s\n", b.Password)
	}
	fmt.Fprintf(&sb, "  Invite    %s\n\n", b.Invite)
	_, _ = io.WriteString(w, sb.String())
	_ = discovery.WriteQR(w, b.Invite)
	_, _ = io.WriteString(w, "\n")
}

func newLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func randomPassword() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)), nil
}

func isLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// advertiseIP picks an address to put in the invite. It prefers an explicit
// listen address and otherwise selects the first private LAN address.
func advertiseIP(listenHost string) string {
	if listenHost != "" {
		if ip := net.ParseIP(listenHost); ip != nil {
			if !ip.IsUnspecified() {
				return ip.String()
			}
		} else {
			return listenHost
		}
	}
	if ip := primaryLANIP(); ip != nil {
		return ip.String()
	}
	return "127.0.0.1"
}

func primaryLANIP() net.IP {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var fallback net.IP
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			ip4 := ip.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}
			if ip4.IsPrivate() {
				return ip4
			}
			if fallback == nil {
				fallback = ip4
			}
		}
	}
	return fallback
}
