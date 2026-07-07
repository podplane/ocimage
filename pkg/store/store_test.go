// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// TestTagSameRepositoryAddsRefAnnotation verifies local tags are represented as OCI ref annotations.
func TestTagSameRepositoryAddsRefAnnotation(t *testing.T) {
	st := Store{Root: t.TempDir()}
	src := name.MustParseReference("zot.local/team/app:v1", name.WeakValidation).(name.Tag)
	dst := name.MustParseReference("zot.local/team/app:latest", name.WeakValidation).(name.Tag)
	if err := st.PutImage(context.Background(), src, empty.Image); err != nil {
		t.Fatal(err)
	}
	if err := st.Tag(context.Background(), src, dst); err != nil {
		t.Fatal(err)
	}
	lp, err := layout.FromPath(filepath.Join(st.Root, "zot.local", "team", "app"))
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
	seen := map[string]bool{}
	for _, desc := range m.Manifests {
		seen[desc.Annotations[RefNameAnnotation]] = true
	}
	if !seen["v1"] || !seen["latest"] {
		t.Fatalf("expected v1 and latest tags, got %v", seen)
	}
}

// TestSaveWritesDockerArchive verifies saved images can be consumed as docker save archives.
func TestSaveWritesDockerArchive(t *testing.T) {
	st := Store{Root: t.TempDir()}
	ref := name.MustParseReference("zot.local/team/app:v1", name.WeakValidation).(name.Tag)
	if err := st.PutImage(context.Background(), ref, empty.Image); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := st.Save(context.Background(), []name.Tag{ref}, nil, &buf); err != nil {
		t.Fatal(err)
	}
	manifest, err := tarball.LoadManifest(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest) != 1 {
		t.Fatalf("manifest entries = %d, want 1", len(manifest))
	}
	if got := manifest[0].RepoTags; len(got) != 1 || got[0] != ref.Name() {
		t.Fatalf("repo tags = %v, want [%s]", got, ref.Name())
	}
}

// TestRepoPathOmitsImplicitDefaultRegistry verifies weak refs do not persist
// go-containerregistry's parser-injected default registry in the store path.
func TestRepoPathOmitsImplicitDefaultRegistry(t *testing.T) {
	dir := t.TempDir()
	st := Store{Root: dir}
	ref := name.MustParseReference("apps/api:latest", name.WeakValidation).(name.Tag)
	if got, want := st.RepoPath(ref), filepath.Join(dir, "apps", "api"); got != want {
		t.Fatalf("RepoPath = %q, want %q", got, want)
	}
	other := name.MustParseReference("ghcr.io/acme/api:latest", name.WeakValidation).(name.Tag)
	if got, want := st.RepoPath(other), filepath.Join(dir, "ghcr.io", "acme", "api"); got != want {
		t.Fatalf("RepoPath other registry = %q, want %q", got, want)
	}
}

// TestLoadDockerArchivesImportsPlatformIndex verifies Docker archives can form an OCI index.
func TestLoadDockerArchivesImportsPlatformIndex(t *testing.T) {
	dir := t.TempDir()
	st := Store{Root: filepath.Join(dir, "store")}
	srcAMD := name.MustParseReference("podplane-build-temp.local/apps/api:v1-amd64", name.WeakValidation).(name.Tag)
	srcARM := name.MustParseReference("podplane-build-temp.local/apps/api:v1-arm64", name.WeakValidation).(name.Tag)
	dst := name.MustParseReference("apps/api:v1", name.WeakValidation).(name.Tag)
	amdPath := filepath.Join(dir, "amd64.tar")
	armPath := filepath.Join(dir, "arm64.tar")
	if err := tarball.WriteToFile(amdPath, srcAMD, empty.Image); err != nil {
		t.Fatal(err)
	}
	if err := tarball.WriteToFile(armPath, srcARM, empty.Image); err != nil {
		t.Fatal(err)
	}
	if err := st.LoadDockerArchives(context.Background(), dst, []PlatformArchive{
		{Path: amdPath, Src: srcAMD, Platform: v1.Platform{OS: "linux", Architecture: "amd64"}},
		{Path: armPath, Src: srcARM, Platform: v1.Platform{OS: "linux", Architecture: "arm64"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(st.Root, "apps", "api", "index.json")); err != nil {
		t.Fatalf("expected imported index: %v", err)
	}
	if _, err := st.Image(context.Background(), dst, v1.Platform{OS: "linux", Architecture: "arm64"}); err != nil {
		t.Fatalf("load arm64 image from imported index: %v", err)
	}
}
