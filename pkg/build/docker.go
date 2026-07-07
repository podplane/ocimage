// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/podplane/ocimage/pkg/containerfile"
	"github.com/podplane/ocimage/pkg/store"
)

var execLookPath = exec.LookPath

var execCommandContext = exec.CommandContext

const dockerNotFoundError = `%s uses Dockerfile syntax ocimage cannot build directly:

  %s

ocimage can use Docker to build this image, but ` + "`docker`" + ` was not found on your PATH.

Either simplify the Containerfile or Dockerfile for ocimage's built-in builder,
or install Docker and rerun the build`

// dockerBuildxFallback builds unsupported Dockerfile syntax with Docker Buildx.
func dockerBuildxFallback(ctx context.Context, opts Options, ctxDir string, file string, unsupported *containerfile.UnsupportedError) (*Result, error) {
	docker, err := resolveDocker(opts.Docker, unsupported)
	if err != nil {
		return nil, err
	}
	if err := ensureDockerBuildx(ctx, docker); err != nil {
		return nil, err
	}
	opts.Docker = docker
	platforms, err := parsePlatforms(opts.Platforms)
	if err != nil {
		return nil, err
	}
	tags, err := parseTags(opts.Tags)
	if err != nil {
		return nil, err
	}
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	tmpTagName := tempDockerTag(opts.Tags[0], hex.EncodeToString(random))
	tmpTag, err := name.NewTag(tmpTagName, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	archives := make([]store.PlatformArchive, 0, len(platforms))
	for _, platform := range platforms {
		archive, err := os.CreateTemp("", "ocimage-build-*.tar")
		if err != nil {
			return nil, err
		}
		archivePath := archive.Name()
		if err := archive.Close(); err != nil {
			_ = os.Remove(archivePath)
			return nil, err
		}
		defer func() { _ = os.Remove(archivePath) }()
		if err := runDockerBuildx(ctx, opts, ctxDir, file, platform, tmpTagName, archivePath); err != nil {
			return nil, err
		}
		archives = append(archives, store.PlatformArchive{Path: archivePath, Src: tmpTag, Platform: platform})
	}
	st := store.Store{Root: opts.StoreRoot}
	if err := st.LoadDockerArchives(ctx, tags[0], archives); err != nil {
		return nil, err
	}
	for _, tag := range tags[1:] {
		if err := st.Tag(ctx, tags[0], tag); err != nil {
			return nil, err
		}
	}
	res := &Result{Tags: opts.Tags, Platforms: platforms}
	if opts.SBOM {
		for i, tag := range tags {
			if opts.Progress != nil {
				opts.Progress("generate sbom for " + opts.Tags[i])
			}
			if err := attachSBOM(ctx, st, tag); err != nil {
				return nil, err
			}
			res.SBOMs = append(res.SBOMs, opts.Tags[i])
		}
	}
	if opts.Push {
		for _, tag := range tags {
			if err := st.Push(ctx, tag); err != nil {
				return nil, err
			}
			res.Pushed = append(res.Pushed, tag.Name())
		}
	}
	return res, nil
}

// resolveDocker resolves the Docker binary used for fallback builds.
func resolveDocker(docker string, unsupported *containerfile.UnsupportedError) (string, error) {
	resolved, err := execLookPath(docker)
	if err != nil {
		return "", fmt.Errorf(dockerNotFoundError, buildFileName(unsupported), unsupported.Error())
	}
	return resolved, nil
}

// ensureDockerBuildx verifies Docker Buildx command is available for fallback builds.
func ensureDockerBuildx(ctx context.Context, docker string) error {
	cmd := execCommandContext(ctx, docker, "buildx", "version")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker was found, but `docker buildx` is not available; Buildx is required for Docker fallback: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// runDockerBuildx writes one platform image as a Docker archive.
func runDockerBuildx(ctx context.Context, opts Options, ctxDir string, file string, platform v1.Platform, tag string, archive string) error {
	args := []string{"buildx", "build", "--file", file, "--tag", tag, "--platform", platform.String(), "--output", "type=docker,dest=" + archive}
	for key, value := range opts.BuildArgs {
		args = append(args, "--build-arg", key+"="+value)
	}
	for key, value := range opts.Labels {
		args = append(args, "--label", key+"="+value)
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	args = append(args, ctxDir)
	if opts.Progress != nil {
		opts.Progress("docker buildx " + platform.String())
	}
	cmd := execCommandContext(ctx, opts.Docker, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker buildx build failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// parseTags parses build tags for store writes.
func parseTags(in []string) ([]name.Tag, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("ocimage build requires at least one tag (-t, --tag)")
	}
	out := make([]name.Tag, 0, len(in))
	for _, tag := range in {
		ref, err := name.NewTag(tag, name.WeakValidation)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

// tempDockerTag returns a recognizable temporary tag for Docker archive output.
func tempDockerTag(storeTag string, random string) string {
	repo, tag := splitRepoTag(storeTag)
	return "ocimage-build-temp.local/" + repo + ":" + tag + "-" + random
}

// splitRepoTag splits an image path into repository and tag components.
func splitRepoTag(pathTag string) (string, string) {
	slash := strings.LastIndex(pathTag, "/")
	colon := strings.LastIndex(pathTag, ":")
	if colon > slash {
		return pathTag[:colon], pathTag[colon+1:]
	}
	return pathTag, "latest"
}

// buildFileName returns a user-facing build file name from an unsupported error.
func buildFileName(unsupported *containerfile.UnsupportedError) string {
	if unsupported == nil || unsupported.Path == "" {
		return "Containerfile or Dockerfile"
	}
	if strings.HasSuffix(unsupported.Path, "Dockerfile") {
		return "Dockerfile"
	}
	if strings.HasSuffix(unsupported.Path, "Containerfile") {
		return "Containerfile"
	}
	return "Containerfile or Dockerfile"
}
