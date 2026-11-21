package discovery

import (
	"fmt"

	"github.com/libp2p/zeroconf/v2"
)

// ServiceType is the mDNS service used by Munna hosts.
const ServiceType = "_munna-proxy._tcp"

// Advertiser publishes a host on the local network.
type Advertiser struct {
	server *zeroconf.Server
}

// Advertise registers instance on the local network with the given TXT
// records. Call Close to stop advertising.
func Advertise(instance string, port int, txt []string) (*Advertiser, error) {
	if instance == "" {
		instance = "munna"
	}
	server, err := zeroconf.Register(instance, ServiceType, "local.", port, txt, nil)
	if err != nil {
		return nil, fmt.Errorf("discovery: mdns register: %w", err)
	}
	return &Advertiser{server: server}, nil
}

// Close stops advertising.
func (a *Advertiser) Close() {
	if a == nil || a.server == nil {
		return
	}
	a.server.Shutdown()
}
