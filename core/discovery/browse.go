package discovery

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/zeroconf/v2"
)

// Peer is a Munna host found on the local network.
type Peer struct {
	Name    string `json:"name"`
	HostID  string `json:"hostID"`
	Address string `json:"address"` // host:port
}

// Browse collects hosts advertised on the local network for the given
// duration. It returns whatever was found; an empty result is not an error.
func Browse(ctx context.Context, timeout time.Duration) ([]Peer, error) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)
	errs := make(chan error, 1)
	go func() {
		errs <- zeroconf.Browse(ctx, ServiceType, "local.", entries)
	}()

	found := map[string]Peer{}
	for {
		select {
		case entry := <-entries:
			peer, ok := peerFromEntry(entry)
			if ok {
				found[peer.Address] = peer
			}
		case err := <-errs:
			if err != nil && ctx.Err() == nil {
				return nil, fmt.Errorf("discovery: browse: %w", err)
			}
			return sortedPeers(found), nil
		case <-ctx.Done():
			return sortedPeers(found), nil
		}
	}
}

func peerFromEntry(entry *zeroconf.ServiceEntry) (Peer, bool) {
	if entry == nil || entry.Port <= 0 {
		return Peer{}, false
	}
	var ip net.IP
	switch {
	case len(entry.AddrIPv4) > 0:
		ip = entry.AddrIPv4[0]
	case len(entry.AddrIPv6) > 0:
		ip = entry.AddrIPv6[0]
	default:
		return Peer{}, false
	}
	peer := Peer{
		Name:    entry.Instance,
		Address: net.JoinHostPort(ip.String(), strconv.Itoa(entry.Port)),
	}
	for _, text := range entry.Text {
		key, value, ok := strings.Cut(text, "=")
		if !ok {
			continue
		}
		switch key {
		case "id":
			peer.HostID = value
		case "name":
			if value != "" {
				peer.Name = value
			}
		}
	}
	return peer, true
}

func sortedPeers(found map[string]Peer) []Peer {
	peers := make([]Peer, 0, len(found))
	for _, peer := range found {
		peers = append(peers, peer)
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].Address < peers[j].Address })
	return peers
}
