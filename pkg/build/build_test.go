// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"archive/tar"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/podplane/ocimage/pkg/store"
)

// TestBuildScratchWritesZotCompatibleStore verifies scratch builds produce Zot-compatible OCI layout output.
func TestBuildScratchWritesZotCompatibleStore(t *testing.T) {
	ctx := context.Background()
	work := t.TempDir()
	storeRoot := filepath.Join(work, "store")
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(filepath.Join(ctxDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "bin", "app"), []byte("#!/app\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte(`FROM scratch
ARG TARGETOS
ARG TARGETARCH
COPY bin/app /app
ENTRYPOINT ["/app"]
`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(ctx, Options{
		ContextDir: ctxDir,
		Tags:       []string{"ghcr.io/octocat/app:v1"},
		Platforms:  []string{"linux/amd64"},
		StoreRoot:  storeRoot,
	})
	if err != nil {
		t.Fatal(err)
	}

	repoPath := filepath.Join(storeRoot, "ghcr.io", "octocat", "app")
	for _, name := range []string{"oci-layout", "index.json", "blobs"} {
		if _, err := os.Stat(filepath.Join(repoPath, name)); err != nil {
			t.Fatalf("expected zot repo layout entry %s: %v", name, err)
		}
	}

	lp, err := layout.FromPath(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := lp.ImageIndex()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := idx.IndexManifest()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(manifest.Manifests); got != 1 {
		t.Fatalf("manifest count = %d, want 1", got)
	}
	if got := manifest.Manifests[0].Annotations[store.RefNameAnnotation]; got != "v1" {
		t.Fatalf("ref annotation = %q, want v1", got)
	}
	img, err := lp.Image(manifest.Manifests[0].Digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := validate.Image(img); err != nil {
		t.Fatal(err)
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Config.User != "65532:65532" {
		t.Fatalf("scratch user = %q, want 65532:65532", cfg.Config.User)
	}
	if len(cfg.Config.Entrypoint) != 1 || cfg.Config.Entrypoint[0] != "/app" {
		b, _ := json.Marshal(cfg.Config.Entrypoint)
		t.Fatalf("entrypoint = %s, want [\"/app\"]", b)
	}
}

// TestBuildUnqualifiedTagOmitsImplicitDefaultRegistry verifies weak local tags
// do not create index.docker.io directories in the OCI store.
func TestBuildUnqualifiedTagOmitsImplicitDefaultRegistry(t *testing.T) {
	work := t.TempDir()
	storeRoot := filepath.Join(work, "store")
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"apps/api:v1"}, StoreRoot: storeRoot}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(storeRoot, "apps", "api", "index.json")); err != nil {
		t.Fatalf("expected apps/api layout: %v", err)
	}
	if _, err := os.Stat(filepath.Join(storeRoot, "index.docker.io")); !os.IsNotExist(err) {
		t.Fatalf("index.docker.io should not be created, stat err = %v", err)
	}
}

// TestBuildParsesQuotedEnvLabelAndCopyMetadata verifies quoted metadata and COPY flags.
func TestBuildParsesQuotedEnvLabelAndCopyMetadata(t *testing.T) {
	ctx := context.Background()
	work := t.TempDir()
	storeRoot := filepath.Join(work, "store")
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(filepath.Join(ctxDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "bin", "app"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte(`FROM scratch
ENV MESSAGE="hello world"
LABEL description="hello world"
COPY --chmod=755 --chown=65532:65532 bin/app /opt/bin/
CMD echo $MESSAGE
`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(ctx, Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, Platforms: []string{"linux/amd64"}, StoreRoot: storeRoot})
	if err != nil {
		t.Fatal(err)
	}
	img := storedImage(t, storeRoot, "ghcr.io/octocat/app:v1")
	cfg, err := img.ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cfg.Config.Env, "MESSAGE=hello world") {
		t.Fatalf("env = %#v, want quoted value preserved", cfg.Config.Env)
	}
	if cfg.Config.Labels["description"] != "hello world" {
		t.Fatalf("label = %q, want quoted value preserved", cfg.Config.Labels["description"])
	}
	if got := cfg.Config.Cmd; len(got) != 3 || got[2] != "echo $MESSAGE" {
		t.Fatalf("cmd = %#v, want shell-form command without build-time expansion", got)
	}

	layers, err := img.Layers()
	if err != nil {
		t.Fatal(err)
	}
	if len(layers) != 1 {
		t.Fatalf("layers = %d, want 1", len(layers))
	}
	rc, err := layers[0].Uncompressed()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	tr := tar.NewReader(rc)
	hdr := findTarHeader(t, tr, "opt/bin/app")
	if hdr.Uid != 65532 || hdr.Gid != 65532 || hdr.Mode != 0o755 {
		t.Fatalf("tar metadata uid=%d gid=%d mode=%#o", hdr.Uid, hdr.Gid, hdr.Mode)
	}
}

// TestBuildRejectsCopyOutsideContext verifies COPY cannot read outside the build context.
func TestBuildRejectsCopyOutsideContext(t *testing.T) {
	work := t.TempDir()
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "secret"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\nCOPY ../secret /secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: filepath.Join(work, "store")})
	if err == nil {
		t.Fatal("expected context escape error")
	}
}

// TestBuildCopiesDirectoryContentsAndHonorsDockerignore verifies directory copy and ignore behavior.
func TestBuildCopiesDirectoryContentsAndHonorsDockerignore(t *testing.T) {
	work := t.TempDir()
	storeRoot := filepath.Join(work, "store")
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(filepath.Join(ctxDir, "dir", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "dir", "app"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "dir", "ignored"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "dir", "sub", "nested"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, ".dockerignore"), []byte("dir/ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\nCOPY dir /app/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: storeRoot})
	if err != nil {
		t.Fatal(err)
	}
	entries := layerEntries(t, storedImage(t, storeRoot, "ghcr.io/octocat/app:v1"))
	if !entries["app/app"] {
		t.Fatalf("expected directory contents under app/, got %v", entries)
	}
	if !entries["app/sub/"] || !entries["app/sub/nested"] {
		t.Fatalf("expected nested directory entries, got %v", entries)
	}
	if entries["app/ignored"] || entries["app/dir/app"] {
		t.Fatalf("unexpected ignored or basename-prefixed entry, got %v", entries)
	}
}

// TestBuildHonorsDockerfileSpecificDockerignore verifies Dockerfile-specific ignore files override the root ignore file.
func TestBuildHonorsDockerfileSpecificDockerignore(t *testing.T) {
	work := t.TempDir()
	storeRoot := filepath.Join(work, "store")
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(filepath.Join(ctxDir, "docker"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "keep"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, ".dockerignore"), []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "docker", "Containerfile"), []byte("FROM scratch\nCOPY keep /keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "docker", "Containerfile.dockerignore"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(context.Background(), Options{ContextDir: ctxDir, File: filepath.Join("docker", "Containerfile"), Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: storeRoot})
	if err != nil {
		t.Fatal(err)
	}
	entries := layerEntries(t, storedImage(t, storeRoot, "ghcr.io/octocat/app:v1"))
	if !entries["keep"] {
		t.Fatalf("expected Dockerfile-specific ignore file to override root ignore, got %v", entries)
	}
}

// TestBuildDockerignoreNegationReincludesChild verifies negation patterns can re-include files below ignored directories.
func TestBuildDockerignoreNegationReincludesChild(t *testing.T) {
	work := t.TempDir()
	storeRoot := filepath.Join(work, "store")
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(filepath.Join(ctxDir, "dir", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "dir", "drop"), []byte("drop"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "dir", "sub", "keep"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, ".dockerignore"), []byte("dir\n!dir/sub\n!dir/sub/keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\nCOPY dir /out/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: storeRoot})
	if err != nil {
		t.Fatal(err)
	}
	entries := layerEntries(t, storedImage(t, storeRoot, "ghcr.io/octocat/app:v1"))
	if !entries["out/sub/keep"] {
		t.Fatalf("expected negated child to be copied, got %v", entries)
	}
	if entries["out/drop"] {
		t.Fatalf("expected ignored sibling to remain excluded, got %v", entries)
	}
}

// TestBuildRejectsMultipleCopySourcesWithoutDirectoryDestination verifies Docker-compatible multi-source validation.
func TestBuildRejectsMultipleCopySourcesWithoutDirectoryDestination(t *testing.T) {
	work := t.TempDir()
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "a"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "b"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\nCOPY a b /dest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: filepath.Join(work, "store")})
	if err == nil {
		t.Fatal("expected multiple source destination error")
	}
}

// TestBuildReportsMissingCopySourceFriendly verifies missing COPY sources produce friendly errors.
func TestBuildReportsMissingCopySourceFriendly(t *testing.T) {
	work := t.TempDir()
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\nCOPY missing /missing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: filepath.Join(work, "store")})
	if err == nil || !strings.Contains(err.Error(), "source \"missing\" not found in build context") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestBuildUsesLocalStoreBase verifies builds can resolve a base image without network access.
func TestBuildUsesLocalStoreBase(t *testing.T) {
	work := t.TempDir()
	ctxDir := filepath.Join(work, "ctx")
	storeRoot := filepath.Join(work, "store")
	baseRef := name.MustParseReference("ghcr.io/base/runtime:v1", name.WeakValidation).(name.Tag)
	if err := (store.Store{Root: storeRoot}).PutImage(context.Background(), baseRef, empty.Image); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM ghcr.io/base/runtime:v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: storeRoot}); err != nil {
		t.Fatal(err)
	}
}

// TestBuildSBOMRequiresSyft verifies SBOM generation fails clearly when syft is absent.
func TestBuildSBOMRequiresSyft(t *testing.T) {
	work := t.TempDir()
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(work, "empty-bin"))
	_, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: filepath.Join(work, "store"), SBOM: true})
	if err == nil || !strings.Contains(err.Error(), "--sbom requires syft") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestBuildSBOMStoresReferrer verifies Syft output is stored as a subject referrer artifact.
func TestBuildSBOMStoresReferrer(t *testing.T) {
	work := t.TempDir()
	ctxDir := filepath.Join(work, "ctx")
	storeRoot := filepath.Join(work, "store")
	binDir := filepath.Join(work, "bin")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "syft"), []byte("#!/bin/sh\nprintf '{\"spdxVersion\":\"SPDX-2.3\"}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	if _, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: storeRoot, SBOM: true}); err != nil {
		t.Fatal(err)
	}
	lp, err := layout.FromPath(filepath.Join(storeRoot, "ghcr.io", "octocat", "app"))
	if err != nil {
		t.Fatal(err)
	}
	idx, err := lp.ImageIndex()
	if err != nil {
		t.Fatal(err)
	}
	m, err := idx.IndexManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Manifests) != 2 {
		t.Fatalf("manifest count = %d, want image and SBOM artifact", len(m.Manifests))
	}
	var subject v1.Descriptor
	for _, desc := range m.Manifests {
		if desc.Annotations[store.RefNameAnnotation] == "v1" {
			subject = desc
		}
	}
	for _, desc := range m.Manifests {
		if desc.Annotations[store.RefNameAnnotation] != "" {
			continue
		}
		if desc.ArtifactType != "application/spdx+json" {
			t.Fatalf("artifact type = %q, want application/spdx+json", desc.ArtifactType)
		}
		img, err := idx.Image(desc.Digest)
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := img.Manifest()
		if err != nil {
			t.Fatal(err)
		}
		if manifest.Subject == nil || manifest.Subject.Digest != subject.Digest {
			t.Fatalf("artifact subject = %#v, want %s", manifest.Subject, subject.Digest)
		}
		if manifest.Layers[0].MediaType != types.MediaType("application/spdx+json") {
			t.Fatalf("layer media type = %s", manifest.Layers[0].MediaType)
		}
		return
	}
	t.Fatal("SBOM artifact not found")
}

// TestBuildScratchSupportsNonrootChown verifies scratch images support the default nonroot identity by name.
func TestBuildScratchSupportsNonrootChown(t *testing.T) {
	work := t.TempDir()
	storeRoot := filepath.Join(work, "store")
	ctxDir := filepath.Join(work, "ctx")
	if err := os.MkdirAll(ctxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "app"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ctxDir, "Containerfile"), []byte("FROM scratch\nCOPY --chown=nonroot:nonroot app /app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), Options{ContextDir: ctxDir, Tags: []string{"ghcr.io/octocat/app:v1"}, StoreRoot: storeRoot}); err != nil {
		t.Fatal(err)
	}
	layers, err := storedImage(t, storeRoot, "ghcr.io/octocat/app:v1").Layers()
	if err != nil {
		t.Fatal(err)
	}
	rc, err := layers[0].Uncompressed()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	hdr := findTarHeader(t, tar.NewReader(rc), "app")
	if hdr.Uid != 65532 || hdr.Gid != 65532 {
		t.Fatalf("tar metadata uid=%d gid=%d, want 65532:65532", hdr.Uid, hdr.Gid)
	}
}

// storedImage loads the first tagged image from the test OCI store.
func storedImage(t *testing.T, storeRoot, ref string) v1.Image {
	t.Helper()
	parts := strings.Split(ref, ":")
	repo := parts[0]
	lp, err := layout.FromPath(filepath.Join(storeRoot, filepath.FromSlash(repo)))
	if err != nil {
		t.Fatal(err)
	}
	idx, err := lp.ImageIndex()
	if err != nil {
		t.Fatal(err)
	}
	m, err := idx.IndexManifest()
	if err != nil {
		t.Fatal(err)
	}
	img, err := lp.Image(m.Manifests[0].Digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := validate.Image(img); err != nil {
		t.Fatal(err)
	}
	return img
}

// layerEntries returns all tar entry names from an image layers.
func layerEntries(t *testing.T, img v1.Image) map[string]bool {
	t.Helper()
	layers, err := img.Layers()
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]bool{}
	for _, layer := range layers {
		rc, err := layer.Uncompressed()
		if err != nil {
			t.Fatal(err)
		}
		tr := tar.NewReader(rc)
		for {
			hdr, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				_ = rc.Close()
				t.Fatal(err)
			}
			entries[hdr.Name] = true
		}
		if err := rc.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return entries
}

// findTarHeader returns the named tar header or fails the test.
func findTarHeader(t *testing.T, tr *tar.Reader, name string) *tar.Header {
	t.Helper()
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				t.Fatalf("tar entry %q not found", name)
			}
			t.Fatal(err)
		}
		if hdr.Name == name {
			return hdr
		}
	}
}
