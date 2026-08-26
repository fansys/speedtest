package sealedbox

import (
	"errors"
	"testing"
)

func TestSealUnsealRoundtrip(t *testing.T) {
	secretKey := "my-very-strong-secret-key-123456"
	plaintext := "node-key-to-protect-abcdef123456"

	sealed, err := Seal(secretKey, plaintext)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}
	if sealed == plaintext {
		t.Fatal("sealed text should not match plaintext")
	}

	unsealed, err := Unseal(secretKey, sealed)
	if err != nil {
		t.Fatalf("Unseal failed: %v", err)
	}
	if unsealed != plaintext {
		t.Fatalf("Unseal got %q, want %q", unsealed, plaintext)
	}
}

func TestUnsealWrongKeyFails(t *testing.T) {
	key1 := "secret-key-number-one-1234567890"
	key2 := "secret-key-number-two-1234567890"
	plaintext := "node-key-abcdef"

	sealed, err := Seal(key1, plaintext)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	_, err = Unseal(key2, sealed)
	if err == nil {
		t.Fatal("expected Unseal to fail with wrong key")
	}
	if !errors.Is(err, ErrSealed) {
		t.Fatalf("expected ErrSealed, got %v", err)
	}
}

func TestUnsealTamperedCiphertext(t *testing.T) {
	secretKey := "my-secret-key-1234567890123456"
	plaintext := "test-message"

	sealed, err := Seal(secretKey, plaintext)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	// 篡改密文
	tampered := []byte(sealed)
	if len(tampered) > 10 {
		if tampered[10] == 'A' {
			tampered[10] = 'B'
		} else {
			tampered[10] = 'A'
		}
	}

	_, err = Unseal(secretKey, string(tampered))
	if err == nil {
		t.Fatal("expected Unseal to fail with tampered ciphertext")
	}
}
