package socks5

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/arafsk/munna-proxy/core/dialer"
)

// Dialer opens outbound connections on behalf of clients.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Config configures a Server.
type Config struct {
	// ListenAddr is used by ListenAndServe and Listen. Defaults to
	// 127.0.0.1:1080.
	ListenAddr string
	// Authenticator enables username/password authentication when non-nil.
	Authenticator Authenticator
	// AuthLimiter throttles failed authentication attempts. It defaults to a
	// per-remote limiter whenever Authenticator is set.
	AuthLimiter *AuthLimiter
	// Dialer is required.
	Dialer Dialer
	// Logger defaults to slog.Default.
	Logger *slog.Logger
	// HandshakeTimeout bounds the greeting, authentication and request
	// phases. Defaults to 10s.
	HandshakeTimeout time.Duration
	// IdleTimeout closes connections with no traffic. Defaults to 5m.
	IdleTimeout time.Duration
	// MaxConnections caps concurrent client connections. Defaults to 512.
	MaxConnections int
}

// Server is a SOCKS5 proxy server.
type Server struct {
	cfg Config

	limiter *AuthLimiter

	connsMu sync.Mutex
	conns   map[net.Conn]struct{}
	closed  bool

	sem chan struct{}
	wg  sync.WaitGroup
}

// New validates cfg and returns a Server.
func New(cfg Config) (*Server, error) {
	if cfg.Dialer == nil {
		return nil, errors.New("socks5: Dialer is required")
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:1080"
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 10 * time.Second
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 5 * time.Minute
	}
	if cfg.MaxConnections <= 0 {
		cfg.MaxConnections = 512
	}
	limiter := cfg.AuthLimiter
	if limiter == nil && cfg.Authenticator != nil {
		limiter = NewAuthLimiter(0, 0, 0)
	}
	return &Server{
		cfg:     cfg,
		limiter: limiter,
		conns:   make(map[net.Conn]struct{}),
		sem:     make(chan struct{}, cfg.MaxConnections),
	}, nil
}

// Listen binds the configured address without serving.
func (s *Server) Listen() (net.Listener, error) {
	return s.ListenContext(context.Background())
}

// ListenContext binds the configured address without serving, aborting if ctx
// is cancelled first.
func (s *Server) ListenContext(ctx context.Context) (net.Listener, error) {
	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(ctx, "tcp", s.cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("socks5: listen: %w", err)
	}
	return ln, nil
}

// ListenAndServe binds the configured address and serves until ctx is done.
func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := s.Listen()
	if err != nil {
		return err
	}
	return s.Serve(ctx, ln)
}

// Serve accepts connections on ln until ctx is done or the listener fails.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	var closeOnce sync.Once
	closeListener := func() {
		closeOnce.Do(func() {
			_ = ln.Close()
		})
	}

	done := make(chan struct{})
	defer close(done)
	defer func() {
		closeListener()
		s.closeAll()
		s.wg.Wait()
	}()

	go func() {
		select {
		case <-ctx.Done():
			closeListener()
		case <-done:
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("socks5: accept: %w", err)
		}

		select {
		case s.sem <- struct{}{}:
		default:
			s.cfg.Logger.Warn("connection limit reached", "remote", remoteHost(conn.RemoteAddr()))
			_ = conn.Close()
			continue
		}
		if !s.track(conn) {
			<-s.sem
			_ = conn.Close()
			continue
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer s.untrack(conn)
			defer func() { <-s.sem }()
			s.handle(ctx, conn)
		}()
	}
}

func (s *Server) track(conn net.Conn) bool {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	if s.closed {
		return false
	}
	s.conns[conn] = struct{}{}
	return true
}

func (s *Server) untrack(conn net.Conn) {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	delete(s.conns, conn)
}

func (s *Server) closeAll() {
	s.connsMu.Lock()
	s.closed = true
	conns := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.connsMu.Unlock()

	for _, conn := range conns {
		_ = conn.Close()
	}
}

func (s *Server) handle(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	remote := remoteHost(conn.RemoteAddr())
	if err := conn.SetDeadline(time.Now().Add(s.cfg.HandshakeTimeout)); err != nil {
		return
	}

	methods, err := readGreeting(conn)
	if err != nil {
		s.cfg.Logger.Debug("greeting failed", "remote", remote, "err", err)
		return
	}

	method := selectMethod(methods, s.cfg.Authenticator != nil)
	if err := writeMethod(conn, method); err != nil {
		return
	}
	if method == methodNoAcceptable {
		s.cfg.Logger.Warn("no acceptable authentication method", "remote", remote)
		return
	}

	if method == methodUserPass {
		username, password, err := readAuth(conn)
		if err != nil {
			s.cfg.Logger.Debug("authentication read failed", "remote", remote, "err", err)
			return
		}
		if s.limiter != nil && !s.limiter.Allow(remote) {
			_ = writeAuthStatus(conn, false)
			s.cfg.Logger.Warn("authentication locked out", "remote", remote)
			return
		}
		ok := s.cfg.Authenticator.Authenticate(ctx, username, password)
		if err := writeAuthStatus(conn, ok); err != nil {
			return
		}
		if !ok {
			if s.limiter != nil {
				s.limiter.Failure(remote)
			}
			s.cfg.Logger.Warn("authentication failed", "remote", remote)
			return
		}
		if s.limiter != nil {
			s.limiter.Success(remote)
		}
	}

	req, err := readRequest(conn)
	if err != nil {
		if errors.Is(err, errUnsupportedAddress) {
			_ = writeReply(conn, repAddressNotSupported, nil)
		}
		s.cfg.Logger.Debug("request failed", "remote", remote, "err", err)
		return
	}

	if req.cmd != cmdConnect {
		_ = writeReply(conn, repCommandNotSupported, nil)
		s.cfg.Logger.Debug("unsupported command", "remote", remote, "command", req.cmd)
		return
	}

	_ = conn.SetDeadline(time.Time{})

	target := net.JoinHostPort(req.host, strconv.Itoa(int(req.port)))
	dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	upstream, err := s.cfg.Dialer.DialContext(dialCtx, "tcp", target)
	if err != nil {
		_ = writeReply(conn, replyForError(err), nil)
		s.cfg.Logger.Debug("dial failed", "remote", remote, "target", target, "err", err)
		return
	}
	defer func() {
		_ = upstream.Close()
	}()

	if err := writeReply(conn, repSuccess, upstream.LocalAddr()); err != nil {
		return
	}
	s.cfg.Logger.Debug("connected", "remote", remote, "target", target)

	relay(conn, upstream, s.cfg.IdleTimeout)
}

func replyForError(err error) byte {
	switch {
	case errors.Is(err, dialer.ErrPolicyDenied):
		return repNotAllowed
	case errors.Is(err, dialer.ErrDNS):
		return repHostUnreachable
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):
		return repTTLExpired
	case errors.Is(err, syscall.ECONNREFUSED):
		return repConnectionRefused
	case errors.Is(err, syscall.ENETUNREACH):
		return repNetworkUnreachable
	case errors.Is(err, syscall.EHOSTUNREACH):
		return repHostUnreachable
	default:
		return repGeneralFailure
	}
}
