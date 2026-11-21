// Command munna-relay is the rendezvous and forwarding server that lets
// hosts and clients reach each other over the internet.
//
// The relay never sees plaintext: clients and hosts run an end-to-end
// handshake inside the bridged connection. See docs/protocol.md.
package main

import (
	"fmt"
	"os"

	"github.com/arafsk/munna-proxy/internal/version"
)

func main() {
	fmt.Printf("munna-relay %s\n", version.String())
	fmt.Fprintln(os.Stderr, "munna-relay: not implemented yet (see docs/roadmap.md)")
	os.Exit(2)
}
