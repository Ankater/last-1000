package db

import "testing"

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_ENV_OR_DEFAULT", "configured")

	if got := envOrDefault("TEST_ENV_OR_DEFAULT", "fallback"); got != "configured" {
		t.Fatalf("expected configured value, got %q", got)
	}
	if got := envOrDefault("TEST_ENV_OR_DEFAULT_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback value, got %q", got)
	}
}
