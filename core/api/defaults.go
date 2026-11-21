package api

import (
	"os"
	"strings"
)

// Defaults are the values a GUI pre-fills in its forms.
type Defaults struct {
	Name         string `json:"name"`
	LANAddress   string `json:"lanAddress"`
	HostListen   string `json:"hostListen"`
	ClientListen string `json:"clientListen"`
	HostPort     string `json:"hostPort"`
}

// DefaultValues reports the local machine name and network defaults.
func DefaultValues() Defaults {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		name = "munna"
	}
	return Defaults{
		Name:         name,
		LANAddress:   AdvertiseIP(""),
		HostListen:   DefaultHostListen,
		ClientListen: DefaultClientListen,
		HostPort:     "1080",
	}
}
