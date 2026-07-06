// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package containerfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseRejectsUnsupportedRun verifies RUN is rejected with line context.
func TestParseRejectsUnsupportedRun(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Containerfile")
	if err := os.WriteFile(file, []byte("FROM scratch\nRUN echo no\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseFile(file)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "RUN is not supported") || !strings.Contains(err.Error(), ":2:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestParseRejectsMultiStage verifies named stages are rejected.
func TestParseRejectsMultiStage(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(file, []byte("FROM scratch AS build\nCOPY app /app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseFile(file)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "multi-stage builds are not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestParseRejectsInstructionBeforeFrom verifies only ARG may precede FROM.
func TestParseRejectsInstructionBeforeFrom(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "Containerfile")
	if err := os.WriteFile(file, []byte("ENV FOO=bar\nFROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseFile(file)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "ENV cannot appear before FROM") {
		t.Fatalf("unexpected error: %v", err)
	}
}
