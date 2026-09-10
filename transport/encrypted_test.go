package transport

import (
	"bytes"
	"testing"
)

type mockTransport struct {
	BaseTransport
	sent [][]byte
}

func (m *mockTransport) Send(data []byte) error {
	m.sent = append(m.sent, append([]byte(nil), data...))
	m.CallReceive(data)
	return nil
}

func TestEncryptedTransport(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	mock := &mockTransport{BaseTransport: *NewBaseTransport(DefaultConfig())}

	encTrans, err := NewEncryptedTransport(mock, key)
	if err != nil {
		t.Fatalf("NewEncryptedTransport failed: %v", err)
	}

	pipeline := NewCompressedTransport(encTrans)

	originalData := []byte("Hello, OpenFlux E2E Encrypted World! Testing compression + encryption pipeline.")
	var received []byte

	pipeline.Receive(func(data []byte) {
		received = data
	})

	if err := pipeline.Send(originalData); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if !bytes.Equal(originalData, received) {
		t.Fatalf("Data mismatch: expected %q, got %q", string(originalData), string(received))
	}

	// Verify that the sent data on the wire is encrypted (does not contain plaintext)
	if bytes.Contains(mock.sent[0], []byte("OpenFlux")) {
		t.Fatalf("Plaintext leak detected in encrypted wire payload!")
	}
}
