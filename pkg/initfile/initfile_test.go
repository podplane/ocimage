// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package initfile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestDetectGoProject verifies a single Go project match can be generated without flags.
func TestDetectGoProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	matches, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Template.ID != "go-scratch-chi" {
		t.Fatalf("matches = %#v, want go-scratch-chi", matches)
	}
}

// TestDetectCSharpWebProject verifies ASP.NET SDK projects select the chiseled template.
func TestDetectCSharpWebProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.csproj"), []byte(`<Project Sdk="Microsoft.NET.Sdk.Web"></Project>`), 0o644); err != nil {
		t.Fatal(err)
	}
	matches, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Template.ID != "csharp-aspnet-chiseled" {
		t.Fatalf("matches = %#v, want csharp-aspnet-chiseled", matches)
	}
}

// TestDetectTypeScriptRuntimeAmbiguous verifies runtime variants are returned when no lockfile chooses one.
func TestDetectTypeScriptRuntimeAmbiguous(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"hono":"latest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	matches, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	ids := matchIDs(matches)
	if !slices.Contains(ids, "typescript-hono-bun") || !slices.Contains(ids, "typescript-hono-node") {
		t.Fatalf("ids = %v, want bun and node Hono variants", ids)
	}
}

// TestWriteTemplateWritesEmbeddedFiles verifies embedded template contents are written to disk.
func TestWriteTemplateWritesEmbeddedFiles(t *testing.T) {
	dir := t.TempDir()
	tmpl, ok := FindTemplate("typescript-tanstack-start-spa")
	if !ok {
		t.Fatal("template not found")
	}
	written, err := Write(dir, tmpl, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(written, "Containerfile") || !slices.Contains(written, "Caddyfile") {
		t.Fatalf("written = %v, want Containerfile and Caddyfile", written)
	}
	body, err := os.ReadFile(filepath.Join(dir, "Containerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "reg.mini.dev/caddy") {
		t.Fatalf("unexpected Containerfile: %s", body)
	}
	if _, err := Write(dir, tmpl, false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite error, got %v", err)
	}
}

// matchIDs returns the template IDs from matches.
func matchIDs(matches []Match) []string {
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match.Template.ID)
	}
	return ids
}
