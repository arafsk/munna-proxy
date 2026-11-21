// Package dialer owns every outbound connection made by the host.
//
// It resolves names with the host's resolver, applies policy, and races
// addresses using a happy-eyeballs strategy so dual-stack targets connect
// quickly.
package dialer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"time"
)

// ErrDNS is returned when a target name cannot be resolved.
var ErrDNS = errors.New("dns resolution failed")

// Config configures a Dialer.
type Config struct {
	// Resolver defaults to net.DefaultResolver.
	Resolver *net.Resolver
	// Timeout bounds a single connection attempt. Defaults to 10s.
	Timeout time.Duration
	// Policy defaults to DefaultPolicy.
	Policy *Policy
	// Logger defaults to slog.Default.
	Logger *slog.Logger
	// HappyEyeballsDelay is the stagger between address attempts.
	// Defaults to 200ms.
	HappyEyeballsDelay time.Duration
}

// Dialer resolves and dials targets on behalf of proxy clients.
type Dialer struct {
	resolver  *net.Resolver
	timeout   time.Duration
	policy    Policy
	logger    *slog.Logger
	netDialer *net.Dialer
	stagger   time.Duration
}

// New returns a Dialer. Zero-valued fields fall back to safe defaults.
func New(cfg Config) *Dialer {
	if cfg.Resolver == nil {
		cfg.Resolver = net.DefaultResolver
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.HappyEyeballsDelay <= 0 {
		cfg.HappyEyeballsDelay = 200 * time.Millisecond
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	policy := DefaultPolicy()
	if cfg.Policy != nil {
		policy = *cfg.Policy
	}
	return &Dialer{
		resolver: cfg.Resolver,
		timeout:  cfg.Timeout,
		policy:   policy,
		logger:   cfg.Logger,
		netDialer: &net.Dialer{
			Timeout:   cfg.Timeout,
			KeepAlive: 30 * time.Second,
		},
		stagger: cfg.HappyEyeballsDelay,
	}
}

// DialContext connects to address ("host:port") using the host's network.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, portString, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("dial %q: %w", address, err)
	}
	port, err := strconv.ParseUint(portString, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("dial %q: invalid port: %w", address, err)
	}
	destinationPort := uint16(port)

	if addr, parseErr := netip.ParseAddr(host); parseErr == nil {
		addr = addr.Unmap()
		if err := d.policy.Check(addr, destinationPort); err != nil {
			return nil, err
		}
		return d.dialOne(ctx, network, netip.AddrPortFrom(addr, destinationPort))
	}

	ips, err := d.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrDNS, host, err)
	}

	addrs := make([]netip.AddrPort, 0, len(ips))
	seen := make(map[netip.Addr]struct{}, len(ips))
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok {
			continue
		}
		addr = addr.Unmap()
		if _, duplicate := seen[addr]; duplicate {
			continue
		}
		seen[addr] = struct{}{}
		if err := d.policy.Check(addr, destinationPort); err != nil {
			d.logger.Debug("resolved address denied", "host", host, "addr", addr, "reason", err)
			continue
		}
		addrs = append(addrs, netip.AddrPortFrom(addr, destinationPort))
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrPolicyDenied, host)
	}
	return d.race(ctx, network, addrs)
}

type dialResult struct {
	conn net.Conn
	err  error
}

func (d *Dialer) dialOne(ctx context.Context, network string, addr netip.AddrPort) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	return d.netDialer.DialContext(ctx, network, addr.String())
}

func (d *Dialer) race(ctx context.Context, network string, addrs []netip.AddrPort) (net.Conn, error) {
	ctx, cancel := context.WithCancel(ctx)
	results := make(chan dialResult, len(addrs))

	for i, addr := range addrs {
		go func(i int, addr netip.AddrPort) {
			if i > 0 {
				timer := time.NewTimer(time.Duration(i) * d.stagger)
				defer timer.Stop()
				select {
				case <-timer.C:
				case <-ctx.Done():
					results <- dialResult{err: ctx.Err()}
					return
				}
			}
			conn, err := d.dialOne(ctx, network, addr)
			if err == nil {
				select {
				case <-ctx.Done():
					_ = conn.Close()
					results <- dialResult{err: ctx.Err()}
					return
				default:
				}
			}
			results <- dialResult{conn: conn, err: err}
		}(i, addr)
	}

	var firstErr error
	for i := 0; i < len(addrs); i++ {
		result := <-results
		if result.err == nil {
			cancel()
			go drainResults(results, len(addrs)-i-1)
			return result.conn, nil
		}
		if firstErr == nil {
			firstErr = result.err
		}
	}
	cancel()
	return nil, firstErr
}

// drainResults closes connections that finish after a winner was chosen.
func drainResults(results <-chan dialResult, remaining int) {
	for i := 0; i < remaining; i++ {
		result := <-results
		if result.conn != nil {
			_ = result.conn.Close()
		}
	}
}
