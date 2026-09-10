package transport

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"universal-bypass-tool/utils"
)

// EncryptedTransport wraps an underlying Transport with ChaCha20-Poly1305 AEAD encryption.
type EncryptedTransport struct {
	Transport
	aead cipher.AEAD
}

// NewEncryptedTransport creates a new EncryptedTransport.
// keyStr can be a 64-character hex string (32 bytes) or any passphrase (hashed with SHA-256).
// If keyStr is empty, returns inner transport unmodified (unencrypted).
func NewEncryptedTransport(inner Transport, keyStr string) (Transport, error) {
	if keyStr == "" {
		return inner, nil
	}

	var key []byte
	if len(keyStr) == 64 {
		if decoded, err := hex.DecodeString(keyStr); err == nil && len(decoded) == 32 {
			key = decoded
		}
	}
	if key == nil {
		h := sha256.Sum256([]byte(keyStr))
		key = h[:]
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to init chacha20poly1305: %w", err)
	}

	utils.Log("[ENCRYPTED] End-to-End ChaCha20-Poly1305 encryption enabled")
	return &EncryptedTransport{
		Transport: inner,
		aead:      aead,
	}, nil
}

func (e *EncryptedTransport) Send(data []byte) error {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	// Seal appends ciphertext and authentication tag to nonce: [nonce (12B) | ciphertext | tag (16B)]
	sealed := e.aead.Seal(nonce, nonce, data, nil)
	return e.Transport.Send(sealed)
}

func (e *EncryptedTransport) Receive(callback func([]byte)) {
	e.Transport.Receive(func(data []byte) {
		nonceSize := e.aead.NonceSize()
		if len(data) < nonceSize+e.aead.Overhead() {
			utils.Debugf("[ENCRYPTED] Packet too short (%d bytes), dropping", len(data))
			return
		}
		nonce := data[:nonceSize]
		ciphertext := data[nonceSize:]
		plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			utils.Debugf("[ENCRYPTED] Decryption failed (invalid key or corrupted packet): %v", err)
			return
		}
		callback(plaintext)
	})
}
