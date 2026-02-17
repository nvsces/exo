package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nvsces/exo/client"
)

const defaultTunnelPort = "9000"

func RunConnect(args []string) {
	fs := flag.NewFlagSet("exo connect", flag.ExitOnError)
	serverAddr := fs.String("server", "localhost:9000", "server address (host or host:port)")
	subdomain := fs.String("subdomain", "", "requested subdomain name")
	localPort := fs.String("local", "3000", "local port to forward to")
	reconnect := fs.Bool("reconnect", true, "auto-reconnect on disconnect")
	fs.Parse(args)

	if *subdomain == "" {
		fmt.Fprintln(os.Stderr, "error: --subdomain is required")
		fs.Usage()
		os.Exit(1)
	}

	// Auto-append default tunnel port if not specified
	addr := *serverAddr
	if !strings.Contains(addr, ":") {
		addr = addr + ":" + defaultTunnelPort
	}

	c := client.New(addr, *subdomain, *localPort)
	if *reconnect {
		c.RunWithReconnect()
	} else {
		if err := c.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}
