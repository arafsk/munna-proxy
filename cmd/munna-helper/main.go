// Command munna-helper is the privileged helper that creates the TUN device,
// installs routes and configures DNS on behalf of the unprivileged client.
//
// It exposes a narrow local API (unix socket or named pipe) and must never
// execute shell commands. See docs/adr/0003-tun-privileged-helper.md.
package main

import (
	"fmt"
	"os"

	"github.com/arafsk/munna-proxy/internal/version"
)

func main() {
	fmt.Printf("munna-helper %s\n", version.String())
	fmt.Fprintln(os.Stderr, "munna-helper: not implemented yet (see docs/roadmap.md)")
	os.Exit(2)
}
