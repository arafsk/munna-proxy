// Command munna is the Munna Proxy CLI and daemon.
//
// It runs in two modes:
//
//	munna serve     host mode: expose a SOCKS5 endpoint on this device
//	munna connect   client mode: mirror a host's network through a TUN device
//
// The command is intentionally thin; the engine lives in core/.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/arafsk/munna-proxy/internal/version"
)

const usage = `munna — cross-platform SOCKS5 network mirror

Usage:
  munna <command> [flags]

Commands:
  serve     Run the host: expose a SOCKS5 endpoint on this device
  connect   Run the client: mirror a host's network via a TUN device
  expose    Publish a host-local port to paired clients
  status    Show the state of a running daemon
  version   Print version information

Run "munna serve -h" for host options. Client mode and the relay arrive in
later milestones; see docs/roadmap.md.
`

func main() {
	os.Exit(run())
}

// run returns the process exit code. Keeping os.Exit out of it lets deferred
// cleanup (signal handling) run before the process terminates.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	var err error
	switch os.Args[1] {
	case "serve":
		err = runServe(ctx, os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println(version.String())
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "munna: %q is not implemented yet (see docs/roadmap.md)\n", os.Args[1])
		return 2
	}

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "munna: %v\n", err)
		return 1
	}
	return 0
}
