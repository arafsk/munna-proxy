package socks5

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
)

// Authenticator verifies SOCKS5 username/password credentials.
type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) bool
}

// StaticAuth verifies a single username/password pair.
//
// Only SHA-256 digests of the credentials are kept in memory, and comparisons
// are constant time.
type StaticAuth struct {
	username [32]byte
	password [32]byte
}

// NewStaticAuth returns an Authenticator for the given credentials.
func NewStaticAuth(username, password string) (*StaticAuth, error) {
	if username == "" || len(username) > 255 {
		return nil, errors.New("socks5: username must be 1-255 bytes")
	}
	if password == "" || len(password) > 255 {
		return nil, errors.New("socks5: password must be 1-255 bytes")
	}
	return &StaticAuth{
		username: sha256.Sum256([]byte(username)),
		password: sha256.Sum256([]byte(password)),
	}, nil
}

// Authenticate implements Authenticator.
func (a *StaticAuth) Authenticate(_ context.Context, username, password string) bool {
	userDigest := sha256.Sum256([]byte(username))
	passDigest := sha256.Sum256([]byte(password))
	userOK := subtle.ConstantTimeCompare(userDigest[:], a.username[:])
	passOK := subtle.ConstantTimeCompare(passDigest[:], a.password[:])
	return userOK&passOK == 1
}
