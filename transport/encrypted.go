package transport

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
	"universal-bypass-tool/utils"
)

// EncryptedTransport wraps an underlying Transport with ChaCha20-Poly1305 AEAD encryption.
type EncryptedTransport struct {
	Transport
	aead cipher.AEAD
}

// NewEncryptedTransport creates a new EncryptedTransport.
// keyStr must be a 64-character hex string (32 bytes = 256 bits).
// Weak human passwords/passphrases without KDF are strictly rejected to prevent offline dictionary brute-force attacks.
// Generate a secure 256-bit key via: openssl rand -hex 32
// If keyStr is empty, returns inner transport unmodified (unencrypted).
func NewEncryptedTransport(inner Transport, keyStr string) (Transport, error) {
	keyStr = strings.TrimSpace(keyStr)
	if keyStr == "" {
		return inner, nil
	}

	if len(keyStr) != 64 {
		return nil, fmt.Errorf("invalid secret key length (%d characters): key must be a 64-character hex string (32 bytes). Weak passwords are not permitted. Generate with: openssl rand -hex 32", len(keyStr))
	}

	key, err := hex.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid secret key (contains non-hex characters): %w. Key must be 64 hexadecimal characters [0-9a-f]", err)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to init chacha20poly1305: %w", err)
	}

	utils.Log("[ENCRYPTED] End-to-End ChaCha20-Poly1305 encryption enabled (256-bit key)")
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
