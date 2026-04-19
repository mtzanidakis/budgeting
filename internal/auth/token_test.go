package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIToken(t *testing.T) {
	raw, hash, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken: %v", err)
	}
	if !strings.HasPrefix(raw, APITokenPrefix) {
		t.Errorf("token missing prefix %q: %s", APITokenPrefix, raw)
	}
	if len(raw) < len(APITokenPrefix)+32 {
		t.Errorf("token too short: %s", raw)
	}
	if hash == "" || hash == raw {
		t.Errorf("hash invalid: %s", hash)
	}
	if HashAPIToken(raw) != hash {
		t.Errorf("HashAPIToken not deterministic")
	}

	raw2, _, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("GenerateAPIToken 2: %v", err)
	}
	if raw == raw2 {
		t.Errorf("tokens should be unique")
	}
}

func TestIsAPITokenFormat(t *testing.T) {
	if !IsAPITokenFormat("bdg_abc") {
		t.Errorf("expected true for bdg_ prefix")
	}
	if IsAPITokenFormat("bdg_") {
		t.Errorf("expected false for prefix-only")
	}
	if IsAPITokenFormat("nope") {
		t.Errorf("expected false for non-prefix")
	}
}
