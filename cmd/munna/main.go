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
	"fmt"
	"os"

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

The proxy engine is not implemented yet. See docs/roadmap.md.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(version.String())
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "munna: %q is not implemented yet (see docs/roadmap.md)\n", os.Args[1])
		os.Exit(2)
	}
}
