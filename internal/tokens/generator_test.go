package tokens

import (
	"encoding/hex"
	"testing"
)

func TestGenerateTokenLengthAndEncoding(t *testing.T) {
	token, err := GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if len(token) != 64 {
		t.Fatalf("expected token length 64, got %d", len(token))
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("expected valid hex token, got %q: %v", token, err)
	}
}

func TestGenerateTokenZeroBytes(t *testing.T) {
	token, err := GenerateToken(0)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
}

func TestGenerateTokenProducesDifferentValues(t *testing.T) {
	first, err := GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	second, err := GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}

	if first == second {
		t.Fatalf("expected different tokens, got %q and %q", first, second)
	}
}
