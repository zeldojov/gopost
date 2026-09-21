package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	hash, err := Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash == "" {
		t.Fatal("Hash() returned empty hash")
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Errorf("Hash() hash has invalid prefix: %q", hash)
	}

	if !strings.Contains(hash, "m=65536,t=3,p=2") {
		t.Errorf("Hash() hash has incorrect parameters: %q", hash)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("Hash() produced %d parts, want 6: %q", len(parts), hash)
	}

	if parts[4] == "" {
		t.Error("Hash() produced empty salt")
	}

	if parts[5] == "" {
		t.Error("Hash() produced empty hash")
	}
}

func TestHashPasswordDifferentHashes(t *testing.T) {
	password := "CorrectHorseBatteryStaple123!"

	hash1, err := Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	hash2, err := Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("Hash() produced identical hashes for the same password")
	}
}

func TestHashPasswordRandError(t *testing.T) {
	original := randRead
	defer func() {
		randRead = original
	}()

	randRead = func([]byte) (int, error) {
		return 0, errors.New("random error")
	}

	hash, err := Hash("CorrectHorseBatteryStaple123!")

	if err == nil {
		t.Fatal("Hash() error = nil, want error")
	}

	if err.Error() != "random error" {
		t.Errorf("HashPassword() error = %q, want %q", err.Error(), "random error")
	}

	if hash != "" {
		t.Errorf("HashPassword() hash = %q, want empty string", hash)
	}
}
