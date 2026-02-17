package client

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/nvsces/exo/proto"
)

type Client struct {
	serverAddr string
	subdomain  string
	localAddr  string
	conn       net.Conn
	mu         sync.Mutex // protects writes to conn
}

func New(serverAddr, subdomain, localPort string) *Client {
	return &Client{
		serverAddr: serverAddr,
		subdomain:  subdomain,
		localAddr:  "localhost:" + localPort,
	}
}

func (c *Client) Run() error {
	conn, err := net.DialTimeout("tcp", c.serverAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", c.serverAddr, err)
	}
	c.conn = conn
	defer conn.Close()

	// Send Hello
	helloPayload, _ := proto.EncodeJSON(proto.HelloMsg{Subdomain: c.subdomain})
	if err := proto.WriteFrame(conn, proto.Frame{
		Type:    proto.TypeHello,
		Payload: helloPayload,
	}); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	// Read HelloAck or HelloErr
	frame, err := proto.ReadFrame(conn)
	if err != nil {
		return fmt.Errorf("read hello response: %w", err)
	}

	switch frame.Type {
	case proto.TypeHelloAck:
		ack, err := proto.DecodeJSON[proto.HelloAckMsg](frame.Payload)
		if err != nil {
			return fmt.Errorf("decode hello ack: %w", err)
		}
		fmt.Printf("\n  exo tunnel active\n")
		fmt.Printf("  Public:  %s\n", ack.URL)
		fmt.Printf("  Local:   http://%s\n\n", c.localAddr)
	case proto.TypeHelloErr:
		errMsg, _ := proto.DecodeJSON[proto.HelloErrMsg](frame.Payload)
		return fmt.Errorf("server rejected: %s", errMsg.Error)
	default:
		return fmt.Errorf("unexpected frame type: %d", frame.Type)
	}

	// Read loop — receive proxied requests
	c.readLoop()
	return nil
}

func (c *Client) RunWithReconnect() {
	for {
		err := c.Run()
		if err != nil {
			slog.Error("connection lost", "err", err)
		}
		slog.Info("reconnecting in 3 seconds...")
		time.Sleep(3 * time.Second)
	}
}

func (c *Client) sendFrame(f proto.Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return proto.WriteFrame(c.conn, f)
}

type streamBuffer struct {
	buf bytes.Buffer
}

func (c *Client) readLoop() {
	pending := make(map[uint32]*streamBuffer)

	for {
		frame, err := proto.ReadFrame(c.conn)
		if err != nil {
			slog.Info("connection closed", "err", err)
			return
		}

		switch frame.Type {
		case proto.TypeHTTPReq:
			sb, ok := pending[frame.StreamID]
			if !ok {
				sb = &streamBuffer{}
				pending[frame.StreamID] = sb
			}
			sb.buf.Write(frame.Payload)

		case proto.TypeHTTPReqEnd:
			sb, ok := pending[frame.StreamID]
			if !ok {
				continue
			}
			delete(pending, frame.StreamID)
			go c.handleStream(frame.StreamID, sb.buf.Bytes())

		case proto.TypePing:
			c.sendFrame(proto.Frame{Type: proto.TypePong})

		default:
			slog.Warn("unknown frame type", "type", frame.Type)
		}
	}
}
