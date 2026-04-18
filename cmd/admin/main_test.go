package main

import (
	"strings"
	"testing"
)

func TestRunCreateTokenRequiresName(t *testing.T) {
	err := runCreateToken(nil)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected missing name error, got %v", err)
	}
}
