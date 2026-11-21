package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arafsk/munna-proxy/core/api"
	"github.com/arafsk/munna-proxy/core/discovery"
)

func runServe(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	listen := flags.String("listen", api.DefaultHostListen, "address to listen on")
	user := flags.String("user", "munna", "username for SOCKS5 authentication")
	password := flags.String("password", "", "password (generated when empty)")
	noAuth := flags.Bool("no-auth", false, "disable authentication (loopback only)")
	advertise := flags.String("advertise", "", "host:port to advertise in the invite")
	name := flags.String("name", "", "host name shown to clients")
	mdns := flags.Bool("mdns", true, "advertise the host over mDNS")
	debug := flags.Bool("debug", false, "enable debug logging")

	if err := flags.Parse(args); err != nil {
		return err
	}

	controller := api.NewWithOutput(os.Stderr)
	controller.SetDebug(*debug)
	defer controller.Close()

	info, err := controller.StartHost(api.HostOptions{
		Listen:      *listen,
		User:        *user,
		Password:    *password,
		NoAuth:      *noAuth,
		Name:        *name,
		Advertise:   *advertise,
		DisableMDNS: !*mdns,
	})
	if err != nil {
		return err
	}

	printBanner(os.Stdout, info)
	<-ctx.Done()
	return nil
}

func printBanner(w io.Writer, info api.HostInfo) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\nMunna host %q\n", info.Name)
	fmt.Fprintf(&sb, "  Listen    %s\n", info.Listen)
	fmt.Fprintf(&sb, "  Advertise %s\n", info.Advertise)
	if info.NoAuth {
		sb.WriteString("  Auth      disabled (loopback only)\n")
	} else {
		fmt.Fprintf(&sb, "  User      %s\n", info.User)
		fmt.Fprintf(&sb, "  Password  %s\n", info.Password)
	}
	fmt.Fprintf(&sb, "  Invite    %s\n\n", info.Invite)
	_, _ = io.WriteString(w, sb.String())
	_ = discovery.WriteQR(w, info.Invite)
	_, _ = io.WriteString(w, "\n")
}
