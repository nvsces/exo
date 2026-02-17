package server

import (
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	"github.com/nvsces/exo/proto"
)

type Tunnel struct {
	subdomain string
	conn      net.Conn
	mu        sync.Mutex             // protects writes to conn
	streams   map[uint32]chan proto.Frame
	streamsMu sync.Mutex
	streamSeq atomic.Uint32
	done      chan struct{}
}

func NewTunnel(subdomain string, conn net.Conn) *Tunnel {
	return &Tunnel{
		subdomain: subdomain,
		conn:      conn,
		streams:   make(map[uint32]chan proto.Frame),
		done:      make(chan struct{}),
	}
}

func (t *Tunnel) NextStreamID() uint32 {
	return t.streamSeq.Add(1)
}

func (t *Tunnel) RegisterStream(id uint32) chan proto.Frame {
	ch := make(chan proto.Frame, 16)
	t.streamsMu.Lock()
	t.streams[id] = ch
	t.streamsMu.Unlock()
	return ch
}

func (t *Tunnel) UnregisterStream(id uint32) {
	t.streamsMu.Lock()
	delete(t.streams, id)
	t.streamsMu.Unlock()
}

func (t *Tunnel) SendFrame(f proto.Frame) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return proto.WriteFrame(t.conn, f)
}

func (t *Tunnel) ReadLoop() {
	defer close(t.done)
	for {
		frame, err := proto.ReadFrame(t.conn)
		if err != nil {
			slog.Info("tunnel disconnected", "subdomain", t.subdomain, "err", err)
			return
		}
		switch frame.Type {
		case proto.TypeHTTPResp, proto.TypeHTTPRespEnd:
			t.streamsMu.Lock()
			ch, ok := t.streams[frame.StreamID]
			t.streamsMu.Unlock()
			if ok {
				ch <- frame
				if frame.Type == proto.TypeHTTPRespEnd {
					close(ch)
				}
			}
		case proto.TypePong:
			// keepalive response, ignore
		default:
			slog.Warn("unknown frame type from client", "type", frame.Type, "subdomain", t.subdomain)
		}
	}
}

func (t *Tunnel) Close() {
	t.conn.Close()
	// Close all pending streams
	t.streamsMu.Lock()
	for id, ch := range t.streams {
		close(ch)
		delete(t.streams, id)
	}
	t.streamsMu.Unlock()
}

func (t *Tunnel) Done() <-chan struct{} {
	return t.done
}
