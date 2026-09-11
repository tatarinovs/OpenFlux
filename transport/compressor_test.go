package transport

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pierrec/lz4/v4"
)

func TestCompressorPipeline(t *testing.T) {
	mock := &mockTransport{BaseTransport: *NewBaseTransport(DefaultConfig())}
	compTrans := NewCompressedTransport(mock)

	originalData := []byte("Testing LZ4 compression and decompression pipeline in OpenFlux transport layer. Repeated data: " +
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")

	var received []byte
	compTrans.Receive(func(data []byte) {
		received = data
	})

	if err := compTrans.Send(originalData); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if !bytes.Equal(originalData, received) {
		t.Fatalf("Data mismatch: expected %d bytes, got %d bytes", len(originalData), len(received))
	}
}

func TestCorruptedDecompressionDropped(t *testing.T) {
	mock := &mockTransport{BaseTransport: *NewBaseTransport(DefaultConfig())}
	compTrans := NewCompressedTransport(mock)

	received := false
	compTrans.Receive(func(data []byte) {
		received = true
	})

	// Send an invalid/corrupted compressed packet (starts with marker 0x1F but contains garbage)
	corrupted := []byte{CompressionMarker, 0xFF, 0xFE, 0xFD, 0x00}
	mock.CallReceive(corrupted)

	if received {
		t.Fatalf("Corrupted packet was NOT dropped; it was passed to callback!")
	}
}

func TestInvalidMarkerRejected(t *testing.T) {
	invalid := []byte{0x77, 0x01, 0x02}
	_, err := decompress(invalid)
	if !errors.Is(err, ErrInvalidMarker) {
		t.Fatalf("Expected ErrInvalidMarker, got %v", err)
	}
}

func TestDecompressionBombProtection(t *testing.T) {
	// Create a payload that expands beyond 10 MB
	bombData := make([]byte, MaxDecompressedSize+2048)
	var buf bytes.Buffer
	buf.WriteByte(CompressionMarker)
	w := lz4.NewWriter(&buf)
	w.Write(bombData)
	w.Close()

	_, err := decompress(buf.Bytes())
	if !errors.Is(err, ErrDecompressionBomb) {
		t.Fatalf("Expected ErrDecompressionBomb, got: %v", err)
	}
}
