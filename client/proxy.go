package client

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/nvsces/exo/proto"
)

func (c *Client) handleStream(streamID uint32, reqBytes []byte) {
	// Parse HTTP request from raw bytes
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(reqBytes)))
	if err != nil {
		slog.Error("failed to parse request", "stream", streamID, "err", err)
		c.sendErrorResponse(streamID, http.StatusBadGateway, "bad request from tunnel")
		return
	}
	defer req.Body.Close()

	// Rewrite URL to point to local service
	req.URL.Scheme = "http"
	req.URL.Host = c.localAddr
	req.RequestURI = "" // required for http.Client

	slog.Info("proxying", "method", req.Method, "path", req.URL.Path, "stream", streamID)

	// Forward to local service
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("local service error", "stream", streamID, "err", err)
		c.sendErrorResponse(streamID, http.StatusBadGateway, fmt.Sprintf("local service error: %v", err))
		return
	}
	defer resp.Body.Close()

	// Serialize response
	var buf bytes.Buffer
	if err := resp.Write(&buf); err != nil {
		slog.Error("failed to serialize response", "stream", streamID, "err", err)
		c.sendErrorResponse(streamID, http.StatusInternalServerError, "failed to serialize response")
		return
	}

	// Send response back through tunnel in chunks
	data := buf.Bytes()
	for offset := 0; offset < len(data); offset += proto.MaxPayloadLen {
		end := offset + proto.MaxPayloadLen
		if end > len(data) {
			end = len(data)
		}
		if err := c.sendFrame(proto.Frame{
			Type:     proto.TypeHTTPResp,
			StreamID: streamID,
			Payload:  data[offset:end],
		}); err != nil {
			slog.Error("failed to send response", "stream", streamID, "err", err)
			return
		}
	}

	// Send end marker
	c.sendFrame(proto.Frame{
		Type:     proto.TypeHTTPRespEnd,
		StreamID: streamID,
	})
}

func (c *Client) sendErrorResponse(streamID uint32, status int, msg string) {
	resp := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Length: %d\r\nContent-Type: text/plain\r\n\r\n%s",
		status, http.StatusText(status), len(msg), msg)

	c.sendFrame(proto.Frame{
		Type:     proto.TypeHTTPResp,
		StreamID: streamID,
		Payload:  []byte(resp),
	})
	c.sendFrame(proto.Frame{
		Type:     proto.TypeHTTPRespEnd,
		StreamID: streamID,
	})
}
