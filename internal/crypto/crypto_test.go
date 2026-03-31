package crypto_test

import (
	"bytes"
	"testing"

	"clippy/internal/crypto"
)

func TestDeriveKey_Length(t *testing.T) {
	key, err := crypto.DeriveKey([]byte("a-secret-that-is-long-enough-yes!!"))
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(key))
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	secret := []byte("a-secret-that-is-long-enough-yes!!")
	key1, err := crypto.DeriveKey(secret)
	if err != nil {
		t.Fatal(err)
	}
	key2, err := crypto.DeriveKey(secret)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key1, key2) {
		t.Error("DeriveKey should be deterministic — same secret must produce same key")
	}
}

func TestDeriveKey_DifferentSecrets(t *testing.T) {
	key1, _ := crypto.DeriveKey([]byte("secret-one-that-is-long-enough-yes!!"))
	key2, _ := crypto.DeriveKey([]byte("secret-two-that-is-long-enough-yes!!"))
	if bytes.Equal(key1, key2) {
		t.Error("different secrets must produce different keys")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key, _ := crypto.DeriveKey([]byte("a-secret-that-is-long-enough-yes!!"))
	plaintext := []byte(`func main() { fmt.Println("hello, clippy") }`)

	ciphertext, err := crypto.Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_UniqueEachTime(t *testing.T) {
	key, _ := crypto.DeriveKey([]byte("a-secret-that-is-long-enough-yes!!"))
	plaintext := []byte("same content every time")

	c1, _ := crypto.Encrypt(key, plaintext)
	c2, _ := crypto.Encrypt(key, plaintext)

	if bytes.Equal(c1, c2) {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1, _ := crypto.DeriveKey([]byte("correct-secret-long-enough-yes!!!!"))
	key2, _ := crypto.DeriveKey([]byte("wrong-secret-that-is-long-enough!!"))

	ciphertext, _ := crypto.Encrypt(key1, []byte("secret content"))

	_, err := Decrypt(key2, ciphertext)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecrypt_TruncatedData(t *testing.T) {
	key, _ := crypto.DeriveKey([]byte("a-secret-that-is-long-enough-yes!!"))

	_, err := Decrypt(key, []byte("tooshort"))
	if err == nil {
		t.Error("expected error for truncated ciphertext")
	}
}

func TestDecrypt_EmptyData(t *testing.T) {
	key, _ := crypto.DeriveKey([]byte("a-secret-that-is-long-enough-yes!!"))

	_, err := Decrypt(key, []byte{})
	if err == nil {
		t.Error("expected error for empty ciphertext")
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	key, _ := crypto.DeriveKey([]byte("a-secret-that-is-long-enough-yes!!"))

	ciphertext, err := crypto.Encrypt(key, []byte{})
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, []byte{}) {
		t.Errorf("expected empty plaintext, got %q", decrypted)
	}
}
