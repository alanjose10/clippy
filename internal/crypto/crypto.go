package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const contextInfo = "clippy-v1-snippet-content"

// DeriveKey produces a 32 byte AES key from the secret
func DeriveKey(secret []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, secret, nil, []byte(contextInfo))

	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	return key, nil
}

// Encrypt encrypts the plain text data to cypher text
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("creating gcm: %w", err)
	}

	// 12 byte random value for nonce
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("creating nonce: %w", err)
	}

	res := gcm.Seal(nonce, nonce, plaintext, nil)

	return res, nil
}

// Decrypt decrypts the data to plain text
func Decrypt(key []byte, data []byte) ([]byte, error) {
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("creating gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()

	// return error if data is too short
	if len(data) < nonceSize {
		return nil, fmt.Errorf("data truncated")
	}

	// output len = len(data) - len(nonce) - 16 (GCM tag)
	size := len(data) - nonceSize - 16
	dst := make([]byte, 0, size)
	dst, err = gcm.Open(
		dst,
		data[:nonceSize],
		data[nonceSize:],
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return dst, nil
}
