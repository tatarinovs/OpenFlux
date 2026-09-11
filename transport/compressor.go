package transport

import (
	"bytes"
	"errors"
	"io"

	"github.com/pierrec/lz4/v4"
)

var (
	ErrDecompressionBomb = errors.New("decompression limit exceeded (potential decompression bomb)")
	ErrInvalidMarker    = errors.New("invalid compression marker")
	ErrEmptyPacket      = errors.New("empty packet")
)

const (
	MinCompressSize     = 200
	CompressionMarker   = 0x1F
	MaxDecompressedSize = 10 * 1024 * 1024 // 10 MB limit to prevent decompression bombs
)

type CompressedTransport struct {
	Transport
}

func NewCompressedTransport(inner Transport) Transport {
	return &CompressedTransport{Transport: inner}
}

func (c *CompressedTransport) Send(data []byte) error {
	compressed := compress(data)
	return c.Transport.Send(compressed)
}

func (c *CompressedTransport) Receive(callback func([]byte)) {
	c.Transport.Receive(func(data []byte) {
		decompressed, err := decompress(data)
		if err != nil {
			// Drop corrupted, truncated or invalid packets immediately.
			// Do NOT forward raw compressed bytes to upper tunnel layer.
			return
		}
		callback(decompressed)
	})
}

func compress(data []byte) []byte {
	if len(data) <= MinCompressSize {
		out := make([]byte, 1, len(data)+1)
		out[0] = 0x00
		out = append(out, data...)
		return out
	}

	var buf bytes.Buffer
	buf.WriteByte(CompressionMarker)

	w := lz4.NewWriter(&buf)
	w.Write(data)
	w.Close()

	if buf.Len() >= len(data)+1 {
		out := make([]byte, 1, len(data)+1)
		out[0] = 0x00
		out = append(out, data...)
		return out
	}

	return buf.Bytes()
}

func decompress(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, ErrEmptyPacket
	}

	if data[0] == 0x00 {
		return data[1:], nil
	}

	if data[0] != CompressionMarker {
		return nil, ErrInvalidMarker
	}

	r := lz4.NewReader(bytes.NewReader(data[1:]))
	// Read up to MaxDecompressedSize + 1 to detect if output exceeds limit
	limitReader := io.LimitReader(r, MaxDecompressedSize+1)
	decompressed, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, err
	}
	if len(decompressed) > MaxDecompressedSize {
		return nil, ErrDecompressionBomb
	}

	return decompressed, nil
}