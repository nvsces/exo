package server

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/nvsces/exo/proto"
)

const streamTimeout = 30 * time.Second

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	subdomain := ExtractSubdomain(r.Host, s.domain)
	if subdomain == "" {
		http.Error(w, "no subdomain specified", http.StatusBadRequest)
		return
	}

	tunnel := s.router.Get(subdomain)
	if tunnel == nil {
		http.Error(w, "tunnel not found: "+subdomain, http.StatusBadGateway)
		return
	}

	streamID := tunnel.NextStreamID()
	respCh := tunnel.RegisterStream(streamID)
	defer tunnel.UnregisterStream(streamID)

	// Serialize the HTTP request
	reqBytes, err := httputil.DumpRequest(r, true)
	if err != nil {
		slog.Error("failed to dump request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Send request in chunks
	for offset := 0; offset < len(reqBytes); offset += proto.MaxPayloadLen {
		end := offset + proto.MaxPayloadLen
		if end > len(reqBytes) {
			end = len(reqBytes)
		}
		if err := tunnel.SendFrame(proto.Frame{
			Type:     proto.TypeHTTPReq,
			StreamID: streamID,
			Payload:  reqBytes[offset:end],
		}); err != nil {
			slog.Error("failed to send request frame", "err", err)
			http.Error(w, "tunnel error", http.StatusBadGateway)
			return
		}
	}

	// Send end-of-request marker
	if err := tunnel.SendFrame(proto.Frame{
		Type:     proto.TypeHTTPReqEnd,
		StreamID: streamID,
	}); err != nil {
		slog.Error("failed to send request end", "err", err)
		http.Error(w, "tunnel error", http.StatusBadGateway)
		return
	}

	// Wait for response
	var respBuf bytes.Buffer
	timeout := time.NewTimer(streamTimeout)
	defer timeout.Stop()

	for {
		select {
		case frame, ok := <-respCh:
			if !ok {
				// Channel closed — tunnel disconnected
				http.Error(w, "tunnel disconnected", http.StatusBadGateway)
				return
			}
			if frame.Type == proto.TypeHTTPRespEnd {
				goto parseResponse
			}
			respBuf.Write(frame.Payload)
		case <-timeout.C:
			http.Error(w, "tunnel timeout", http.StatusGatewayTimeout)
			return
		}
	}

parseResponse:
	resp, err := http.ReadResponse(bufio.NewReader(&respBuf), r)
	if err != nil {
		slog.Error("failed to parse tunneled response", "err", err)
		http.Error(w, "bad response from tunnel", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
