package cmd

import (
	"flag"
	"log/slog"
	"os"

	"github.com/nvsces/exo/server"
)

func RunServer(args []string) {
	fs := flag.NewFlagSet("exo server", flag.ExitOnError)
	domain := fs.String("domain", "localhost", "base domain for subdomains (e.g. example.com)")
	httpPort := fs.String("port", "80", "HTTP listen port")
	ctrlPort := fs.String("tunnel-port", "9000", "tunnel control port")
	fs.Parse(args)

	srv := server.New(*domain, *httpPort, *ctrlPort)
	if err := srv.Run(); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
