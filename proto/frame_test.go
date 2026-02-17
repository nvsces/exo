package proto

import (
	"bytes"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
	}{
		{
			name:  "empty payload",
			frame: Frame{Type: TypePing, StreamID: 0, Payload: nil},
		},
		{
			name:  "hello message",
			frame: Frame{Type: TypeHello, StreamID: 0, Payload: []byte(`{"subdomain":"myapp"}`)},
		},
		{
			name:  "data frame",
			frame: Frame{Type: TypeHTTPReq, StreamID: 42, Payload: []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")},
		},
		{
			name:  "large payload",
			frame: Frame{Type: TypeHTTPResp, StreamID: 100, Payload: bytes.Repeat([]byte("x"), 1024)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteFrame(&buf, tt.frame); err != nil {
				t.Fatalf("WriteFrame: %v", err)
			}

			got, err := ReadFrame(&buf)
			if err != nil {
				t.Fatalf("ReadFrame: %v", err)
			}

			if got.Type != tt.frame.Type {
				t.Errorf("Type: got %d, want %d", got.Type, tt.frame.Type)
			}
			if got.StreamID != tt.frame.StreamID {
				t.Errorf("StreamID: got %d, want %d", got.StreamID, tt.frame.StreamID)
			}
			if !bytes.Equal(got.Payload, tt.frame.Payload) {
				t.Errorf("Payload mismatch: got %d bytes, want %d bytes", len(got.Payload), len(tt.frame.Payload))
			}
		})
	}
}

func TestFrameTooLarge(t *testing.T) {
	f := Frame{Type: TypeHTTPReq, StreamID: 1, Payload: make([]byte, MaxPayloadLen+1)}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, f); err == nil {
		t.Fatal("expected error for oversized payload")
	}
}

func TestMultipleFrames(t *testing.T) {
	var buf bytes.Buffer
	frames := []Frame{
		{Type: TypeHello, StreamID: 0, Payload: []byte("hello")},
		{Type: TypeHTTPReq, StreamID: 1, Payload: []byte("request")},
		{Type: TypeHTTPResp, StreamID: 1, Payload: []byte("response")},
		{Type: TypePing, StreamID: 0},
	}

	for _, f := range frames {
		if err := WriteFrame(&buf, f); err != nil {
			t.Fatalf("WriteFrame: %v", err)
		}
	}

	for i, want := range frames {
		got, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("ReadFrame[%d]: %v", i, err)
		}
		if got.Type != want.Type || got.StreamID != want.StreamID {
			t.Errorf("frame[%d]: got type=%d sid=%d, want type=%d sid=%d",
				i, got.Type, got.StreamID, want.Type, want.StreamID)
		}
		if !bytes.Equal(got.Payload, want.Payload) {
			t.Errorf("frame[%d]: payload mismatch", i)
		}
	}
}
