package main

import (
	"fmt"
	"os"

	"github.com/nvsces/exo/cmd"
)

const usage = `exo — expose local services to the internet

Usage:
  exo server   Start the tunnel server on a VPS
  exo connect  Connect a local port to the server

Server:
  exo server --domain example.com --port 80 --tunnel-port 9000

Client:
  exo connect --server example.com --subdomain myapp --local 3000

This makes http://myapp.example.com forward to localhost:3000
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		cmd.RunServer(os.Args[2:])
	case "connect":
		cmd.RunConnect(os.Args[2:])
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}
