package main

import (
	"strings"
	"testing"
)

func TestExecCmdIncludesStderrOnFailure(t *testing.T) {
	_, err := execCmd("sh", "-c", "echo not-found-detail >&2; exit 7")
	if err == nil {
		t.Fatal("execCmd returned nil error for failing command")
	}
	if !strings.Contains(err.Error(), "not-found-detail") {
		t.Fatalf("execCmd error missing stderr detail: %v", err)
	}
}
