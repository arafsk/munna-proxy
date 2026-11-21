package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/arafsk/munna-proxy/core/dialer"
	"github.com/arafsk/munna-proxy/core/discovery"
	"github.com/arafsk/munna-proxy/core/socks5"
)

// DefaultHostListen is where a host listens unless configured otherwise.
const DefaultHostListen = "0.0.0.0:1080"

// DefaultClientListen is the local SOCKS5 address a client exposes.
const DefaultClientListen = "127.0.0.1:1081"

// Controller owns the host and client sessions for one application instance.
type Controller struct {
	logs  *logBuffer
	level *slog.LevelVar

	mu     sync.Mutex
	logger *slog.Logger
	host   *hostSession
	client *clientSession
}

// New returns a Controller with no sessions running. Logs are kept in memory
// for Logs.
func New() *Controller {
	return NewWithOutput(nil)
}

// NewWithOutput is like New but also writes every log line to out, which is
// useful for the command-line host.
func NewWithOutput(out io.Writer) *Controller {
	logs := &logBuffer{}
	var writer io.Writer = logs
	if out != nil {
		writer = io.MultiWriter(logs, out)
	}
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level}))
	return &Controller{logs: logs, level: level, logger: logger}
}

// SetDebug toggles debug logging for subsequent events.
func (c *Controller) SetDebug(enabled bool) {
	if enabled {
		c.level.Set(slog.LevelDebug)
		return
	}
	c.level.Set(slog.LevelInfo)
}

// Logs returns log entries newer than seq.
func (c *Controller) Logs(seq uint64) []LogEntry {
	return c.logs.since(seq)
}

// HostOptions configures a host session.
type HostOptions struct {
	Listen      string `json:"listen"`    // default DefaultHostListen
	User        string `json:"user"`      // default "munna"
	Password    string `json:"password"`  // generated when empty and auth is enabled
	NoAuth      bool   `json:"noAuth"`    // loopback listeners only
	Name        string `json:"name"`      // default: system hostname
	Advertise   string `json:"advertise"` // host:port written into the invite; default: detected LAN address
	DisableMDNS bool   `json:"disableMDNS"`
}

// HostInfo describes a running or stopped host session.
type HostInfo struct {
	Running   bool      `json:"running"`
	Name      string    `json:"name"`
	Listen    string    `json:"listen"`
	Advertise string    `json:"advertise"`
	User      string    `json:"user"`
	Password  string    `json:"password"`
	NoAuth    bool      `json:"noAuth"`
	MDNS      bool      `json:"mdns"`
	Invite    string    `json:"invite"`
	QR        string    `json:"qr"`
	Started   time.Time `json:"started"`
}

type hostSession struct {
	info       HostInfo
	cancel     context.CancelFunc
	done       chan struct{}
	advertiser *discovery.Advertiser
}

// StartHost binds the listener, advertises the host and serves SOCKS5 until
// StopHost is called.
func (c *Controller) StartHost(opts HostOptions) (HostInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.host != nil {
		return HostInfo{}, errors.New("host is already running")
	}

	listen := opts.Listen
	if listen == "" {
		listen = DefaultHostListen
	}
	host, portString, err := net.SplitHostPort(listen)
	if err != nil {
		return HostInfo{}, fmt.Errorf("invalid listen address: %w", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil || port < 0 || port > 65535 {
		return HostInfo{}, fmt.Errorf("invalid listen port %q", portString)
	}
	if opts.NoAuth && !IsLoopbackHost(host) {
		return HostInfo{}, errors.New("authentication can only be disabled on a loopback address")
	}

	var authenticator socks5.Authenticator
	user := opts.User
	password := opts.Password
	if !opts.NoAuth {
		if user == "" {
			user = "munna"
		}
		if password == "" {
			password, err = RandomSecret()
			if err != nil {
				return HostInfo{}, err
			}
		}
		staticAuth, err := socks5.NewStaticAuth(user, password)
		if err != nil {
			return HostInfo{}, err
		}
		authenticator = staticAuth
	}

	server, err := socks5.New(socks5.Config{
		ListenAddr:    listen,
		Authenticator: authenticator,
		Dialer:        dialer.New(dialer.Config{Logger: c.logger}),
		Logger:        c.logger,
	})
	if err != nil {
		return HostInfo{}, err
	}
	listener, err := server.Listen()
	if err != nil {
		return HostInfo{}, err
	}
	actualPort := port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}

	fail := func(err error) (HostInfo, error) {
		_ = listener.Close()
		return HostInfo{}, err
	}

	hostID, err := discovery.NewHostID()
	if err != nil {
		return fail(err)
	}
	name := opts.Name
	if name == "" {
		name, err = os.Hostname()
		if err != nil || name == "" {
			name = "munna"
		}
	}

	advertised := opts.Advertise
	if advertised == "" {
		advertised = net.JoinHostPort(AdvertiseIP(host), strconv.Itoa(actualPort))
	} else if _, _, err := net.SplitHostPort(advertised); err != nil {
		return fail(fmt.Errorf("invalid advertise address: %w", err))
	}

	invite, err := discovery.Invite{Version: 1, HostID: hostID, Host: advertised, Name: name}.String()
	if err != nil {
		return fail(err)
	}
	qrImage, err := discovery.QRDataURL(invite)
	if err != nil {
		return fail(err)
	}

	session := &hostSession{done: make(chan struct{})}
	if !opts.DisableMDNS {
		advertiser, err := discovery.Advertise(name, actualPort, []string{
			"v=1",
			"id=" + hostID,
			"name=" + name,
		})
		if err != nil {
			c.logger.Warn("mDNS advertisement failed", "err", err)
		} else {
			session.advertiser = advertiser
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	session.cancel = cancel
	session.info = HostInfo{
		Running:   true,
		Name:      name,
		Listen:    listener.Addr().String(),
		Advertise: advertised,
		User:      user,
		Password:  password,
		NoAuth:    opts.NoAuth,
		MDNS:      session.advertiser != nil,
		Invite:    invite,
		QR:        qrImage,
		Started:   time.Now(),
	}
	if opts.NoAuth {
		session.info.User = ""
		session.info.Password = ""
	}

	c.host = session
	c.logger.Info("host listening", "listen", session.info.Listen, "advertise", advertised, "auth", !opts.NoAuth)
	go func() {
		defer close(session.done)
		if err := server.Serve(ctx, listener); err != nil {
			c.logger.Error("host stopped with error", "err", err)
		}
	}()
	return session.info, nil
}

// StopHost stops the host session and waits for connections to close.
func (c *Controller) StopHost() {
	c.mu.Lock()
	session := c.host
	c.host = nil
	c.mu.Unlock()
	if session == nil {
		return
	}
	session.cancel()
	if session.advertiser != nil {
		session.advertiser.Close()
	}
	<-session.done
	c.logger.Info("host stopped")
}

// HostStatus reports the current host session.
func (c *Controller) HostStatus() HostInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.host == nil {
		return HostInfo{}
	}
	return c.host.info
}

// ClientOptions configures a client session.
type ClientOptions struct {
	// Host is a munna:// invite or a host:port address on the local network.
	Host     string `json:"host"`
	User     string `json:"user"` // empty when the host runs with -no-auth
	Password string `json:"password"`
	// Listen is the local SOCKS5 address. It must be a loopback address.
	Listen string `json:"listen"`
}

// ClientInfo describes a running or stopped client session.
type ClientInfo struct {
	Running bool      `json:"running"`
	Host    string    `json:"host"`
	Listen  string    `json:"listen"`
	User    string    `json:"user"`
	Started time.Time `json:"started"`
}

type clientSession struct {
	info   ClientInfo
	cancel context.CancelFunc
	done   chan struct{}
}

// StartClient checks that the host accepts the credentials, then exposes a
// local SOCKS5 endpoint that forwards every connection through the host.
func (c *Controller) StartClient(opts ClientOptions) (ClientInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return ClientInfo{}, errors.New("client is already running")
	}

	hostAddress, err := ResolveHostAddress(opts.Host)
	if err != nil {
		return ClientInfo{}, err
	}
	listen := opts.Listen
	if listen == "" {
		listen = DefaultClientListen
	}
	listenHost, _, err := net.SplitHostPort(listen)
	if err != nil {
		return ClientInfo{}, fmt.Errorf("invalid local address: %w", err)
	}
	if !IsLoopbackHost(listenHost) {
		return ClientInfo{}, errors.New("the local forwarder must listen on a loopback address")
	}

	var credentials *socks5.Credentials
	if opts.User != "" {
		credentials = &socks5.Credentials{Username: opts.User, Password: opts.Password}
	}
	upstream := socks5.Upstream{Address: hostAddress, Credentials: credentials}

	probeCtx, probeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	probe, err := upstream.Authenticate(probeCtx)
	probeCancel()
	if err != nil {
		return ClientInfo{}, fmt.Errorf("cannot reach host %s: %w", hostAddress, err)
	}
	_ = probe.Close()

	server, err := socks5.New(socks5.Config{
		ListenAddr: listen,
		Dialer:     socks5.Chain{Upstream: upstream},
		Logger:     c.logger,
	})
	if err != nil {
		return ClientInfo{}, err
	}
	listener, err := server.Listen()
	if err != nil {
		return ClientInfo{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &clientSession{
		cancel: cancel,
		done:   make(chan struct{}),
		info: ClientInfo{
			Running: true,
			Host:    hostAddress,
			Listen:  listener.Addr().String(),
			User:    opts.User,
			Started: time.Now(),
		},
	}
	c.client = session
	c.logger.Info("client forwarding", "local", session.info.Listen, "host", hostAddress)
	go func() {
		defer close(session.done)
		if err := server.Serve(ctx, listener); err != nil {
			c.logger.Error("client stopped with error", "err", err)
		}
	}()
	return session.info, nil
}

// StopClient stops the local forwarder.
func (c *Controller) StopClient() {
	c.mu.Lock()
	session := c.client
	c.client = nil
	c.mu.Unlock()
	if session == nil {
		return
	}
	session.cancel()
	<-session.done
	c.logger.Info("client stopped")
}

// ClientStatus reports the current client session.
func (c *Controller) ClientStatus() ClientInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return ClientInfo{}
	}
	return c.client.info
}

// Discover lists hosts advertised on the local network.
func (c *Controller) Discover(ctx context.Context, timeout time.Duration) ([]discovery.Peer, error) {
	return discovery.Browse(ctx, timeout)
}

// Close stops every session. It is safe to call more than once.
func (c *Controller) Close() {
	c.StopClient()
	c.StopHost()
}

// ResolveHostAddress turns an invite or host:port into a host:port address.
// Invites without a LAN address cannot be used for direct connections yet.
func ResolveHostAddress(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("enter an invite or a host address")
	}
	if strings.HasPrefix(input, "munna://") {
		invite, err := discovery.ParseInvite(input)
		if err != nil {
			return "", err
		}
		if invite.Host == "" {
			return "", errors.New("this invite has no LAN address; use the host's address instead")
		}
		return invite.Host, nil
	}
	if _, _, err := net.SplitHostPort(input); err != nil {
		return "", fmt.Errorf("host must be host:port: %w", err)
	}
	return input, nil
}

// IsLoopbackHost reports whether host names a loopback interface.
func IsLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// AdvertiseIP picks an address to put in an invite. An explicit listen
// address wins; otherwise the first private IPv4 address is used.
func AdvertiseIP(listenHost string) string {
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
