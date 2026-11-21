package dialer

import (
	"errors"
	"fmt"
	"net/netip"
)

// ErrPolicyDenied is returned when a target is rejected by policy.
var ErrPolicyDenied = errors.New("target denied by policy")

// Policy controls which targets the host is willing to dial.
//
// The zero value denies loopback and metadata endpoints. Use DefaultPolicy
// for the standard host behaviour.
type Policy struct {
	// AllowLoopback permits targets on 127.0.0.0/8 and ::1. This is required
	// for clients to reach services bound to the host's loopback.
	AllowLoopback bool
	// DeniedPorts rejects specific destination ports.
	DeniedPorts map[uint16]struct{}
	// DeniedPrefixes rejects specific address ranges. Metadata endpoints are
	// always included by DefaultPolicy.
	DeniedPrefixes []netip.Prefix
}

// DefaultPolicy returns the policy used by a host unless configured otherwise.
func DefaultPolicy() Policy {
	return Policy{
		AllowLoopback:  true,
		DeniedPorts:    map[uint16]struct{}{},
		DeniedPrefixes: metadataPrefixes(),
	}
}

// metadataPrefixes lists cloud instance metadata endpoints. Reaching these
// through a proxy can leak credentials, so they are denied by default.
func metadataPrefixes() []netip.Prefix {
	raw := []string{
		"169.254.169.254/32", // AWS, GCP, Azure, DigitalOcean, OpenStack
		"169.254.170.2/32",   // AWS ECS task metadata
		"100.100.100.200/32", // Alibaba Cloud
		"192.0.0.192/32",     // Oracle Cloud
		"fd00:ec2::254/128",  // AWS IPv6
	}
	prefixes := make([]netip.Prefix, 0, len(raw))
	for _, value := range raw {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}

// Check reports whether addr:port may be dialed.
func (p Policy) Check(addr netip.Addr, port uint16) error {
	if !addr.IsValid() {
		return fmt.Errorf("%w: invalid address", ErrPolicyDenied)
	}
	addr = addr.Unmap()

	if _, denied := p.DeniedPorts[port]; denied {
		return fmt.Errorf("%w: port %d", ErrPolicyDenied, port)
	}
	if !p.AllowLoopback && addr.IsLoopback() {
		return fmt.Errorf("%w: loopback address", ErrPolicyDenied)
	}
	for _, prefix := range p.DeniedPrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("%w: metadata address %s", ErrPolicyDenied, addr)
		}
	}
	return nil
}
