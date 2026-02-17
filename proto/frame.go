package proto

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	HeaderSize    = 9
	MaxPayloadLen = 64 * 1024 // 64 KB
)

// Message types
const (
	TypeHello       uint8 = 0x01
	TypeHelloAck    uint8 = 0x02
	TypeHelloErr    uint8 = 0x03
	TypeHTTPReq     uint8 = 0x10
	TypeHTTPReqEnd  uint8 = 0x11
	TypeHTTPResp    uint8 = 0x12
	TypeHTTPRespEnd uint8 = 0x13
	TypePing        uint8 = 0x20
	TypePong        uint8 = 0x21
)

type Frame struct {
	Type     uint8
	StreamID uint32
	Payload  []byte
}

func WriteFrame(w io.Writer, f Frame) error {
	if len(f.Payload) > MaxPayloadLen {
		return fmt.Errorf("payload too large: %d > %d", len(f.Payload), MaxPayloadLen)
	}
	var hdr [HeaderSize]byte
	hdr[0] = f.Type
	binary.BigEndian.PutUint32(hdr[1:5], f.StreamID)
	binary.BigEndian.PutUint32(hdr[5:9], uint32(len(f.Payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	if len(f.Payload) > 0 {
		_, err := w.Write(f.Payload)
		return err
	}
	return nil
}

func ReadFrame(r io.Reader) (Frame, error) {
	var hdr [HeaderSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return Frame{}, err
	}
	f := Frame{
		Type:     hdr[0],
		StreamID: binary.BigEndian.Uint32(hdr[1:5]),
	}
	length := binary.BigEndian.Uint32(hdr[5:9])
	if length > MaxPayloadLen {
		return Frame{}, fmt.Errorf("frame too large: %d > %d", length, MaxPayloadLen)
	}
	if length > 0 {
		f.Payload = make([]byte, length)
		if _, err := io.ReadFull(r, f.Payload); err != nil {
			return Frame{}, err
		}
	}
	return f, nil
}
