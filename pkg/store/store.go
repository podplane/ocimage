// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

const RefNameAnnotation = "org.opencontainers.image.ref.name"

const referrerTagPrefix = "sha256-"

type Store struct{ Root string }

// PutImage writes an image to the store under ref.
func (s Store) PutImage(_ context.Context, ref name.Tag, img v1.Image) error {
	lp, err := s.ensureLayout(ref)
	if err != nil {
		return err
	}
	if err := lp.WriteImage(img); err != nil {
		return err
	}
	desc, err := partial.Descriptor(img)
	if err != nil {
		return err
	}
	desc.Annotations = maps.Clone(desc.Annotations)
	if desc.Annotations == nil {
		desc.Annotations = map[string]string{}
	}
	desc.Annotations[RefNameAnnotation] = ref.TagStr()
	return appendDescriptor(lp, *desc)
}

// Image returns the image stored under ref for platform.
func (s Store) Image(_ context.Context, ref name.Tag, platform v1.Platform) (v1.Image, error) {
	lp, desc, err := s.find(ref)
	if err != nil {
		return nil, err
	}
	if !desc.MediaType.IsIndex() {
		return lp.Image(desc.Digest)
	}
	idx, err := lp.ImageIndex()
	if err != nil {
		return nil, err
	}
	child, err := idx.ImageIndex(desc.Digest)
	if err != nil {
		return nil, err
	}
	manifest, err := child.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, candidate := range manifest.Manifests {
		if candidate.Platform != nil && candidate.Platform.Satisfies(platform) {
			return child.Image(candidate.Digest)
		}
	}
	return nil, fmt.Errorf("platform %s not found for %s in ocimage store", platform.String(), ref.Name())
}

// Descriptor returns the store descriptor tagged by ref.
func (s Store) Descriptor(_ context.Context, ref name.Tag) (v1.Descriptor, error) {
	_, desc, err := s.find(ref)
	return desc, err
}

// PutArtifact writes an OCI artifact that refers to subject.
func (s Store) PutArtifact(_ context.Context, ref name.Tag, subject v1.Descriptor, artifact v1.Image, title string) error {
	lp, err := s.ensureLayout(ref)
	if err != nil {
		return err
	}
	artifact = mutate.Annotations(artifact, map[string]string{"org.opencontainers.image.title": title}).(v1.Image)
	artifact = mutate.Subject(artifact, subject).(v1.Image)
	if err := lp.WriteImage(artifact); err != nil {
		return err
	}
	desc, err := partial.Descriptor(artifact)
	if err != nil {
		return err
	}
	return appendDescriptor(lp, *desc)
}

// PutIndex writes an image index to the store under ref.
func (s Store) PutIndex(_ context.Context, ref name.Tag, idx v1.ImageIndex) error {
	lp, err := s.ensureLayout(ref)
	if err != nil {
		return err
	}
	if err := lp.WriteIndex(idx); err != nil {
		return err
	}
	desc, err := partial.Descriptor(idx)
	if err != nil {
		return err
	}
	desc.Annotations = maps.Clone(desc.Annotations)
	if desc.Annotations == nil {
		desc.Annotations = map[string]string{}
	}
	desc.Annotations[RefNameAnnotation] = ref.TagStr()
	return appendDescriptor(lp, *desc)
}

// Push pushes a stored image or index to its remote registry reference.
func (s Store) Push(ctx context.Context, ref name.Tag) error {
	lp, desc, err := s.find(ref)
	if err != nil {
		return err
	}
	opts := []remote.Option{remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain)}
	if desc.MediaType.IsIndex() {
		idx, err := lp.ImageIndex()
		if err != nil {
			return err
		}
		child, err := idx.ImageIndex(desc.Digest)
		if err != nil {
			return err
		}
		if err := remote.WriteIndex(ref, child, opts...); err != nil {
			return err
		}
		return s.pushReferrers(ctx, ref, lp, desc, opts...)
	}
	img, err := lp.Image(desc.Digest)
	if err != nil {
		return err
	}
	if err := remote.Write(ref, img, opts...); err != nil {
		return err
	}
	return s.pushReferrers(ctx, ref, lp, desc, opts...)
}

// Save writes stored images as a Docker-compatible image archive.
func (s Store) Save(ctx context.Context, refs []name.Tag, platform *v1.Platform, w io.Writer) error {
	images := map[name.Reference]v1.Image{}
	for _, ref := range refs {
		img, err := s.imageForSave(ctx, ref, platform)
		if err != nil {
			return err
		}
		images[ref] = img
	}
	return tarball.MultiRefWrite(images, w)
}

// Tag creates dst as another store reference to src.
func (s Store) Tag(ctx context.Context, src, dst name.Tag) error {
	lp, desc, err := s.find(src)
	if err != nil {
		return err
	}
	if s.RepoPath(src) == s.RepoPath(dst) {
		idx, err := readIndexJSON(lp)
		if err != nil {
			return err
		}
		idx.Manifests = withoutTag(idx.Manifests, dst.TagStr())
		desc.Annotations = maps.Clone(desc.Annotations)
		if desc.Annotations == nil {
			desc.Annotations = map[string]string{}
		}
		desc.Annotations[RefNameAnnotation] = dst.TagStr()
		idx.Manifests = append(idx.Manifests, desc)
		return writeIndexJSON(lp, idx)
	}
	if desc.MediaType.IsIndex() {
		idx, err := lp.ImageIndex()
		if err != nil {
			return err
		}
		child, err := idx.ImageIndex(desc.Digest)
		if err != nil {
			return err
		}
		return s.PutIndex(ctx, dst, child)
	}
	img, err := lp.Image(desc.Digest)
	if err != nil {
		return err
	}
	return s.PutImage(ctx, dst, img)
}

// RepoPath returns the Zot-compatible repository layout path for ref.
func (s Store) RepoPath(ref name.Tag) string {
	if ref.Context().RegistryStr() == name.DefaultRegistry {
		return filepath.Join(s.Root, filepath.FromSlash(ref.Context().RepositoryStr()))
	}
	return filepath.Join(s.Root, filepath.FromSlash(ref.Context().Name()))
}

// imageForSave loads the stored image for ref, selecting platform from indexes.
func (s Store) imageForSave(_ context.Context, ref name.Tag, platform *v1.Platform) (v1.Image, error) {
	lp, desc, err := s.find(ref)
	if err != nil {
		return nil, err
	}
	if !desc.MediaType.IsIndex() {
		return lp.Image(desc.Digest)
	}
	if platform == nil {
		return nil, fmt.Errorf("image %s is a multi-platform index; specify --platform", ref.Name())
	}
	idx, err := lp.ImageIndex()
	if err != nil {
		return nil, err
	}
	child, err := idx.ImageIndex(desc.Digest)
	if err != nil {
		return nil, err
	}
	manifest, err := child.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, candidate := range manifest.Manifests {
		if candidate.Platform != nil && candidate.Platform.Satisfies(*platform) {
			return child.Image(candidate.Digest)
		}
	}
	return nil, fmt.Errorf("platform %s not found for %s in ocimage store", platform.String(), ref.Name())
}

// ensureLayout creates or opens the repository OCI layout for ref.
func (s Store) ensureLayout(ref name.Tag) (layout.Path, error) {
	if s.Root == "" {
		return "", errors.New("store root is required")
	}
	repo := s.RepoPath(ref)
	if err := os.MkdirAll(repo, 0o755); err != nil {
		return "", err
	}
	lp, err := layout.FromPath(repo)
	if err == nil {
		return lp, nil
	}
	return layout.Write(repo, empty.Index)
}

// find locates the descriptor tagged by ref in the store.
func (s Store) find(ref name.Tag) (layout.Path, v1.Descriptor, error) {
	lp, err := layout.FromPath(s.RepoPath(ref))
	if err != nil {
		return "", v1.Descriptor{}, fmt.Errorf("image %s not found in ocimage store %s", ref.Name(), s.Root)
	}
	idx, err := lp.ImageIndex()
	if err != nil {
		return "", v1.Descriptor{}, err
	}
	m, err := idx.IndexManifest()
	if err != nil {
		return "", v1.Descriptor{}, err
	}
	for _, desc := range m.Manifests {
		if desc.Annotations[RefNameAnnotation] == ref.TagStr() {
			return lp, desc, nil
		}
	}
	return "", v1.Descriptor{}, fmt.Errorf("tag %s not found in ocimage store repository %s", ref.TagStr(), ref.Context().Name())
}

// pushReferrers pushes locally stored artifacts that refer to subject.
func (s Store) pushReferrers(ctx context.Context, ref name.Tag, lp layout.Path, subject v1.Descriptor, opts ...remote.Option) error {
	idx, err := readIndexJSON(lp)
	if err != nil {
		return err
	}
	root, err := lp.ImageIndex()
	if err != nil {
		return err
	}
	for _, desc := range idx.Manifests {
		if desc.Annotations[RefNameAnnotation] != "" || !desc.MediaType.IsImage() {
			continue
		}
		img, err := root.Image(desc.Digest)
		if err != nil {
			return err
		}
		manifest, err := img.Manifest()
		if err != nil {
			return err
		}
		if manifest.Subject == nil || manifest.Subject.Digest != subject.Digest {
			continue
		}
		if err := remote.Write(referrerTag(ref, desc), img, append(opts, remote.WithContext(ctx))...); err != nil {
			return err
		}
	}
	return nil
}

// referrerTag returns a stable synthetic tag for pushing a referrer manifest.
func referrerTag(ref name.Tag, desc v1.Descriptor) name.Tag {
	return ref.Context().Tag(referrerTagPrefix + strings.TrimPrefix(desc.Digest.String(), "sha256:"))
}

// appendDescriptor atomically updates index.json with desc as the current tag target.
func appendDescriptor(lp layout.Path, desc v1.Descriptor) error {
	idx, err := readIndexJSON(lp)
	if err != nil {
		return err
	}
	idx.Manifests = withoutTag(idx.Manifests, desc.Annotations[RefNameAnnotation])
	idx.Manifests = append(idx.Manifests, desc)
	return writeIndexJSON(lp, idx)
}

// withoutTag removes descriptors with the provided ref-name annotation.
func withoutTag(in []v1.Descriptor, tag string) []v1.Descriptor {
	out := in[:0]
	for _, desc := range in {
		if desc.Annotations[RefNameAnnotation] != tag {
			out = append(out, desc)
		}
	}
	return out
}

// readIndexJSON reads a layout index.json file.
func readIndexJSON(lp layout.Path) (*v1.IndexManifest, error) {
	b, err := os.ReadFile(filepath.Join(string(lp), "index.json"))
	if err != nil {
		return nil, err
	}
	var idx v1.IndexManifest
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// writeIndexJSON atomically writes a layout index.json file.
func writeIndexJSON(lp layout.Path, idx *v1.IndexManifest) error {
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := filepath.Join(string(lp), "index.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
