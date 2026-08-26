package security

import (
	"testing"
)

func TestGenerateNodeKey(t *testing.T) {
	k1 := GenerateNodeKey()
	k2 := GenerateNodeKey()

	if len(k1) < MinNodeKeyLength {
		t.Fatalf("generated key too short: len=%d", len(k1))
	}
	if k1 == k2 {
		t.Fatal("generated keys should be unique")
	}
}

func TestHashAndFingerprint(t *testing.T) {
	key := "test-key-1234567890abcdef"
	hash := HashNodeKey(key)
	if len(hash) != 64 {
		t.Fatalf("expected 64 char hex hash, got len=%d", len(hash))
	}

	fp := KeyFingerprint(key)
	if len(fp) != 12 {
		t.Fatalf("expected 12 char fingerprint, got len=%d", len(fp))
	}
	if fp != hash[:12] {
		t.Fatalf("fingerprint %q does not match prefix of hash %q", fp, hash)
	}
}

func TestConstantTimeEquals(t *testing.T) {
	if !ConstantTimeEquals("secret123", "secret123") {
		t.Fatal("identical strings should be equal")
	}
	if ConstantTimeEquals("secret123", "secret456") {
		t.Fatal("different strings should not be equal")
	}
	if ConstantTimeEquals("secret123", "secret1234") {
		t.Fatal("different length strings should not be equal")
	}
}

func TestPickTokenAndExtractBearer(t *testing.T) {
	if got := ExtractBearer("Bearer mytoken123"); got != "mytoken123" {
		t.Fatalf("got %q, want mytoken123", got)
	}
	if got := ExtractBearer("bearer mytoken123"); got != "mytoken123" {
		t.Fatalf("got %q, want mytoken123", got)
	}
	if got := ExtractBearer("Basic mytoken123"); got != "" {
		t.Fatalf("expected empty for Basic auth, got %q", got)
	}
	if got := ExtractBearer(""); got != "" {
		t.Fatalf("expected empty for empty auth, got %q", got)
	}

	if got := PickToken("headertoken", "Bearer othertoken"); got != "headertoken" {
		t.Fatalf("header should take precedence, got %q", got)
	}
	if got := PickToken("", "Bearer othertoken"); got != "othertoken" {
		t.Fatalf("fallback to bearer, got %q", got)
	}
	if got := PickToken("", ""); got != "" {
		t.Fatalf("both empty, got %q", got)
	}
}
