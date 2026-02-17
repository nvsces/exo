package server

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/nvsces/exo/proto"
)

type Server struct {
	domain   string
	httpAddr string
	ctrlAddr string
	router   *Router
}

func New(domain, httpPort, ctrlPort string) *Server {
	return &Server{
		domain:   domain,
		httpAddr: ":" + httpPort,
		ctrlAddr: ":" + ctrlPort,
		router:   NewRouter(),
	}
}

func (s *Server) Run() error {
	// Start control listener
	ctrlLn, err := net.Listen("tcp", s.ctrlAddr)
	if err != nil {
		return fmt.Errorf("control listen %s: %w", s.ctrlAddr, err)
	}
	defer ctrlLn.Close()
	slog.Info("control listener started", "addr", s.ctrlAddr)

	go s.acceptControlConns(ctrlLn)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    s.httpAddr,
		Handler: s,
	}

	// Graceful shutdown on signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down...")
		httpServer.Close()
		ctrlLn.Close()
	}()

	slog.Info("HTTP server started", "addr", s.httpAddr, "domain", s.domain)
	fmt.Printf("\n  exo server running\n")
	fmt.Printf("  HTTP:    http://%s%s\n", s.domain, s.httpAddr)
	fmt.Printf("  Tunnel:  %s\n\n", s.ctrlAddr)

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

func (s *Server) acceptControlConns(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Debug("control listener closed", "err", err)
			return
		}
		go s.handleControlConn(conn)
	}
}

func (s *Server) handleControlConn(conn net.Conn) {
	defer conn.Close()

	// Read Hello frame
	frame, err := proto.ReadFrame(conn)
	if err != nil {
		slog.Error("failed to read hello", "err", err, "remote", conn.RemoteAddr())
		return
	}
	if frame.Type != proto.TypeHello {
		slog.Error("expected hello frame", "got", frame.Type, "remote", conn.RemoteAddr())
		return
	}

	hello, err := proto.DecodeJSON[proto.HelloMsg](frame.Payload)
	if err != nil {
		slog.Error("failed to decode hello", "err", err)
		return
	}

	if hello.Subdomain == "" {
		s.sendHelloErr(conn, "subdomain is required")
		return
	}

	tunnel := NewTunnel(hello.Subdomain, conn)

	if !s.router.Register(hello.Subdomain, tunnel) {
		s.sendHelloErr(conn, fmt.Sprintf("subdomain %q is already in use", hello.Subdomain))
		return
	}
	defer func() {
		s.router.Unregister(hello.Subdomain)
		tunnel.Close()
		slog.Info("tunnel removed", "subdomain", hello.Subdomain)
	}()

	// Send HelloAck
	url := fmt.Sprintf("http://%s.%s%s", hello.Subdomain, s.domain, s.httpAddr)
	ackPayload, _ := proto.EncodeJSON(proto.HelloAckMsg{URL: url})
	if err := proto.WriteFrame(conn, proto.Frame{
		Type:    proto.TypeHelloAck,
		Payload: ackPayload,
	}); err != nil {
		slog.Error("failed to send hello ack", "err", err)
		return
	}

	slog.Info("tunnel registered", "subdomain", hello.Subdomain, "remote", conn.RemoteAddr())

	// Read loop — dispatch response frames
	tunnel.ReadLoop()
}

func (s *Server) sendHelloErr(conn net.Conn, msg string) {
	payload, _ := proto.EncodeJSON(proto.HelloErrMsg{Error: msg})
	proto.WriteFrame(conn, proto.Frame{
		Type:    proto.TypeHelloErr,
		Payload: payload,
	})
}
