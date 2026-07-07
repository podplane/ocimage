// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/podplane/ocimage/pkg/containerfile"
)

// TestBuildDockerFallbackWithoutDockerExplainsRequirement verifies missing Docker errors are clear.
func TestBuildDockerFallbackWithoutDockerExplainsRequirement(t *testing.T) {
	work := t.TempDir()
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\nRUN echo hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	originalLookPath := execLookPath
	execLookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { execLookPath = originalLookPath })
	_, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"apps/api:v1"}, StoreRoot: filepath.Join(work, "store"), Docker: "docker"})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"RUN is not supported", "docker` was not found"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	}
}

// TestResolveDockerUsesConfiguredBinary verifies --docker path values are resolved directly.
func TestResolveDockerUsesConfiguredBinary(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(name string) (string, error) {
		if name != "/custom/docker" {
			t.Fatalf("look path name = %q, want /custom/docker", name)
		}
		return name, nil
	}
	t.Cleanup(func() { execLookPath = originalLookPath })
	got, err := resolveDocker("/custom/docker", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/docker" {
		t.Fatalf("resolveDocker = %q, want /custom/docker", got)
	}
}

// TestEnsureDockerBuildxRequiresBuildx verifies Docker without Buildx is explained.
func TestEnsureDockerBuildxRequiresBuildx(t *testing.T) {
	originalCommandContext := execCommandContext
	execCommandContext = func(context.Context, string, ...string) *exec.Cmd {
		return exec.Command("go", "definitely-not-a-go-command")
	}
	t.Cleanup(func() { execCommandContext = originalCommandContext })
	err := ensureDockerBuildx(context.Background(), "docker")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "docker buildx") {
		t.Fatalf("error = %q, want buildx", err.Error())
	}
}

// TestTempDockerTagKeepsDesiredRef verifies temporary tags remain identifiable.
func TestTempDockerTagKeepsDesiredRef(t *testing.T) {
	got := tempDockerTag("apps/api:v1", "abc123")
	want := "ocimage-build-temp.local/apps/api:v1-abc123"
	if got != want {
		t.Fatalf("tempDockerTag = %q, want %q", got, want)
	}
}

// TestBuildFileNameReportsKnownBuildFiles verifies fallback errors name the build file.
func TestBuildFileNameReportsKnownBuildFiles(t *testing.T) {
	for _, tt := range []struct {
		path string
		want string
	}{
		{path: "/app/Dockerfile", want: "Dockerfile"},
		{path: "/app/Containerfile", want: "Containerfile"},
		{path: "/app/custom.build", want: "Containerfile or Dockerfile"},
	} {
		got := buildFileName(&containerfile.UnsupportedError{Path: tt.path})
		if got != tt.want {
			t.Fatalf("buildFileName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
