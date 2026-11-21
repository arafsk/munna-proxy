package main

import (
	"context"
	"time"

	"github.com/arafsk/munna-proxy/core/api"
	"github.com/arafsk/munna-proxy/core/discovery"
)

// App is the object bound to the web UI. Every exported method is callable
// from JavaScript; the engine itself lives in core/api.
type App struct {
	ctx        context.Context
	controller *api.Controller
}

// NewApp returns an App with no sessions running.
func NewApp() *App {
	return &App{controller: api.New()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {
	a.controller.Close()
}

// Defaults returns values the forms are pre-filled with.
func (a *App) Defaults() api.Defaults {
	return api.DefaultValues()
}

// StartHost starts a host session on this machine.
func (a *App) StartHost(opts api.HostOptions) (api.HostInfo, error) {
	return a.controller.StartHost(opts)
}

// StopHost stops the host session.
func (a *App) StopHost() {
	a.controller.StopHost()
}

// HostStatus reports the host session.
func (a *App) HostStatus() api.HostInfo {
	return a.controller.HostStatus()
}

// StartClient connects this machine to a host and opens a local proxy.
func (a *App) StartClient(opts api.ClientOptions) (api.ClientInfo, error) {
	return a.controller.StartClient(opts)
}

// StopClient closes the local proxy.
func (a *App) StopClient() {
	a.controller.StopClient()
}

// ClientStatus reports the client session.
func (a *App) ClientStatus() api.ClientInfo {
	return a.controller.ClientStatus()
}

// Discover lists hosts advertising on the local network.
func (a *App) Discover() ([]discovery.Peer, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.controller.Discover(ctx, 3*time.Second)
}

// Logs returns log lines newer than seq.
func (a *App) Logs(seq uint64) []api.LogEntry {
	return a.controller.Logs(seq)
}

// SetDebug enables or disables debug logging.
func (a *App) SetDebug(enabled bool) {
	a.controller.SetDebug(enabled)
}
