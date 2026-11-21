// Package discovery advertises hosts on the local network and encodes
// pairing information for the GUIs.
package discovery

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	inviteScheme  = "munna"
	inviteVersion = 1
)

// Invite is the information a client needs to reach and pair with a host.
type Invite struct {
	Version     int
	HostID      string
	Host        string // host:port, optional when Relay is set
	Relay       string
	Fingerprint string
	Token       string
	Name        string
}

// String encodes the invite as a munna:// URI.
func (i Invite) String() (string, error) {
	if i.HostID == "" {
		return "", errors.New("discovery: invite has no host ID")
	}
	version := i.Version
	if version == 0 {
		version = inviteVersion
	}
	if version != inviteVersion {
		return "", fmt.Errorf("discovery: unsupported invite version %d", version)
	}
	if i.Host != "" {
		if _, _, err := net.SplitHostPort(i.Host); err != nil {
			return "", fmt.Errorf("discovery: invalid host %q: %w", i.Host, err)
		}
	}

	uri := &url.URL{Scheme: inviteScheme}
	if i.Host != "" {
		uri.User = url.User(i.HostID)
		uri.Host = i.Host
	} else {
		uri.Host = i.HostID
	}

	query := url.Values{}
	query.Set("v", strconv.Itoa(version))
	if i.Relay != "" {
		query.Set("relay", i.Relay)
	}
	if i.Fingerprint != "" {
		query.Set("fp", i.Fingerprint)
	}
	if i.Token != "" {
		query.Set("t", i.Token)
	}
	if i.Name != "" {
		query.Set("n", i.Name)
	}
	uri.RawQuery = query.Encode()
	return uri.String(), nil
}

// ParseInvite decodes a munna:// URI.
func ParseInvite(raw string) (Invite, error) {
	uri, err := url.Parse(raw)
	if err != nil {
		return Invite{}, fmt.Errorf("discovery: parse invite: %w", err)
	}
	if uri.Scheme != inviteScheme {
		return Invite{}, fmt.Errorf("discovery: unsupported scheme %q", uri.Scheme)
	}

	invite := Invite{Version: inviteVersion}
	if uri.User != nil {
		invite.HostID = uri.User.Username()
		invite.Host = uri.Host
	} else {
		invite.HostID = uri.Host
	}
	if invite.HostID == "" {
		return Invite{}, errors.New("discovery: invite has no host ID")
	}
	if invite.Host != "" {
		if _, _, err := net.SplitHostPort(invite.Host); err != nil {
			return Invite{}, fmt.Errorf("discovery: invalid host %q: %w", invite.Host, err)
		}
	}

	query := uri.Query()
	if value := query.Get("v"); value != "" {
		version, err := strconv.Atoi(value)
		if err != nil {
			return Invite{}, fmt.Errorf("discovery: invalid version %q", value)
		}
		invite.Version = version
	}
	if invite.Version != inviteVersion {
		return Invite{}, fmt.Errorf("discovery: unsupported invite version %d", invite.Version)
	}

	invite.Relay = query.Get("relay")
	invite.Fingerprint = query.Get("fp")
	invite.Token = query.Get("t")
	invite.Name = query.Get("n")
	return invite, nil
}

// NewHostID returns a random 128-bit host identifier encoded in lowercase
// base32 without padding.
func NewHostID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("discovery: generate host ID: %w", err)
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)), nil
}

// NewToken returns a random single-use pairing token.
func NewToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("discovery: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
