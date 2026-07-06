// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/moby/patternmatcher"
	"github.com/moby/patternmatcher/ignorefile"
	"github.com/podplane/ocimage/pkg/containerfile"
	"github.com/podplane/ocimage/pkg/store"
	"golang.org/x/sync/errgroup"
)

// Build packages the configured context into OCI images and stores or pushes them.
func Build(ctx context.Context, opts Options) (*Result, error) {
	if opts.ContextDir == "" {
		opts.ContextDir = "."
	}
	progress := func(format string, args ...any) {
		if opts.Progress != nil {
			opts.Progress(fmt.Sprintf(format, args...))
		}
	}
	ctxDir, err := filepath.Abs(opts.ContextDir)
	if err != nil {
		return nil, err
	}
	file := opts.File
	if file == "" {
		file, err = defaultFile(ctxDir)
		if err != nil {
			return nil, err
		}
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(ctxDir, file)
	}
	progress("load %s", filepath.Base(file))
	cf, err := containerfile.ParseFile(file)
	if err != nil {
		return nil, err
	}
	matcher, err := dockerignoreMatcher(ctxDir, file)
	if err != nil {
		return nil, err
	}
	platforms, err := parsePlatforms(opts.Platforms)
	if err != nil {
		return nil, err
	}
	if len(opts.Tags) == 0 {
		return nil, fmt.Errorf("ocimage build requires at least one tag (-t, --tag)")
	}
	tags := make([]name.Tag, 0, len(opts.Tags))
	for _, t := range opts.Tags {
		ref, err := name.NewTag(t, name.WeakValidation)
		if err != nil {
			return nil, err
		}
		tags = append(tags, ref)
	}
	st := store.Store{Root: opts.StoreRoot}
	if err := cacheTaggedBases(ctx, st, cf, opts, platforms); err != nil {
		return nil, err
	}
	buildOpts := opts
	buildOpts.Pull = false
	images := make([]v1.Image, len(platforms))
	g, buildCtx := errgroup.WithContext(ctx)
	for i, p := range platforms {
		i, p := i, p
		g.Go(func() error {
			progress("build %s", p.String())
			img, err := buildPlatform(buildCtx, st, ctxDir, cf, buildOpts, p, matcher, true)
			if err != nil {
				return err
			}
			images[i] = img
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	imgs := make([]mutate.IndexAddendum, 0, len(platforms))
	for i, p := range platforms {
		pp := p
		imgs = append(imgs, mutate.IndexAddendum{Add: images[i], Descriptor: v1.Descriptor{Platform: &pp}})
	}
	for _, tag := range tags {
		if len(platforms) == 1 {
			progress("store %s", tag.Name())
			if err := st.PutImage(ctx, tag, images[0]); err != nil {
				return nil, err
			}
		} else {
			progress("store %s", tag.Name())
			idx := mutate.IndexMediaType(mutate.AppendManifests(empty.Index, imgs...), types.OCIImageIndex)
			if err := st.PutIndex(ctx, tag, idx); err != nil {
				return nil, err
			}
		}
	}
	res := &Result{Tags: opts.Tags, Platforms: platforms}
	if opts.SBOM {
		for _, tag := range tags {
			progress("generate sbom for %s", tag.Name())
			if err := attachSBOM(ctx, st, tag); err != nil {
				return nil, err
			}
			res.SBOMs = append(res.SBOMs, tag.Name())
		}
	}
	if opts.Push {
		for _, tag := range tags {
			progress("push %s", tag.Name())
			if err := st.Push(ctx, tag); err != nil {
				return nil, err
			}
			res.Pushed = append(res.Pushed, tag.Name())
		}
	}
	return res, nil
}

type cachedBase struct {
	tag       name.Tag
	platforms []v1.Platform
	images    []v1.Image
	write     bool
}

// cacheTaggedBases resolves tagged remote bases into the local store before platform builds.
func cacheTaggedBases(ctx context.Context, st store.Store, cf *containerfile.File, opts Options, platforms []v1.Platform) error {
	progress := func(format string, args ...any) {
		if opts.Progress != nil {
			opts.Progress(fmt.Sprintf(format, args...))
		}
	}
	bases := map[string]*cachedBase{}
	for _, p := range platforms {
		ref, scratch, err := baseReference(cf, opts.BuildArgs, p)
		if err != nil {
			return err
		}
		if scratch {
			continue
		}
		tag, ok := ref.(name.Tag)
		if !ok {
			continue
		}
		key := tag.Name()
		base := bases[key]
		if base == nil {
			base = &cachedBase{tag: tag}
			bases[key] = base
		}
		base.platforms = append(base.platforms, p)
	}
	for _, base := range bases {
		base.images = make([]v1.Image, len(base.platforms))
		for i, p := range base.platforms {
			if !opts.Pull {
				progress("check local base %s for %s", base.tag.Name(), p.String())
				img, err := st.Image(ctx, base.tag, p)
				if err == nil {
					progress("found local base %s for %s", base.tag.Name(), p.String())
					base.images[i] = img
					continue
				}
				progress("base %s not found locally for %s, downloading", base.tag.Name(), p.String())
			}
			progress("download base %s for %s", base.tag.Name(), p.String())
			img, err := remote.Image(base.tag, remote.WithContext(ctx), remote.WithPlatform(p), remote.WithAuthFromKeychain(authn.DefaultKeychain))
			if err != nil {
				return err
			}
			progress("downloaded base %s for %s", base.tag.Name(), p.String())
			base.images[i] = img
			base.write = true
		}
		if !base.write {
			continue
		}
		progress("cache base %s", base.tag.Name())
		if len(base.images) == 1 {
			if err := st.PutImage(ctx, base.tag, base.images[0]); err != nil {
				return err
			}
			continue
		}
		adds := make([]mutate.IndexAddendum, 0, len(base.images))
		for i, img := range base.images {
			p := base.platforms[i]
			adds = append(adds, mutate.IndexAddendum{Add: img, Descriptor: v1.Descriptor{Platform: &p}})
		}
		idx := mutate.IndexMediaType(mutate.AppendManifests(empty.Index, adds...), types.OCIImageIndex)
		if err := st.PutIndex(ctx, base.tag, idx); err != nil {
			return err
		}
	}
	return nil
}

// baseReference returns the expanded FROM reference for a target platform.
func baseReference(cf *containerfile.File, buildArgs map[string]string, p v1.Platform) (name.Reference, bool, error) {
	args := map[string]string{}
	for k, v := range platformArgs(p) {
		args[k] = v
	}
	for k, v := range buildArgs {
		args[k] = v
	}
	for _, in := range cf.Instructions {
		switch in.Op {
		case "ARG":
			k, v, ok := strings.Cut(in.Args, "=")
			k = strings.TrimSpace(k)
			if _, exists := args[k]; !exists && ok {
				args[k] = expand(strings.TrimSpace(v), args)
			}
		case "FROM":
			if len(in.Tokens) == 0 {
				return nil, false, fmt.Errorf("FROM at line %d requires a base image", in.Line)
			}
			base := expand(in.Tokens[0], args)
			if base == "scratch" {
				return nil, true, nil
			}
			ref, err := name.ParseReference(base, name.WeakValidation)
			return ref, false, err
		}
	}
	return nil, false, fmt.Errorf("missing FROM instruction")
}

// buildPlatform builds one platform-specific image from the parsed Containerfile.
func buildPlatform(ctx context.Context, st store.Store, ctxDir string, cf *containerfile.File, opts Options, p v1.Platform, matcher *patternmatcher.PatternMatcher, basesCached bool) (v1.Image, error) {
	progress := func(format string, args ...any) {
		if opts.Progress != nil {
			opts.Progress(fmt.Sprintf(format, args...))
		}
	}
	args := map[string]string{}
	for k, v := range platformArgs(p) {
		args[k] = v
	}
	for k, v := range opts.BuildArgs {
		args[k] = v
	}
	var img v1.Image
	var cfg *v1.ConfigFile
	needsResolver := needsNamedChown(cf)
	resolver := idResolver{}
	var err error
	for _, in := range cf.Instructions {
		switch in.Op {
		case "ARG":
			k, v, ok := strings.Cut(in.Args, "=")
			k = strings.TrimSpace(k)
			if _, exists := args[k]; !exists && ok {
				args[k] = expand(strings.TrimSpace(v), args)
			}
		case "FROM":
			if len(in.Tokens) == 0 {
				return nil, fmt.Errorf("FROM at line %d requires a base image", in.Line)
			}
			base := expand(in.Tokens[0], args)
			if base == "scratch" {
				progress("base scratch for %s", p.String())
				img = empty.Image
				resolver = idResolver{users: map[string]userEntry{"nonroot": {uid: 65532, gid: 65532}}, groups: map[string]int{"nonroot": 65532}}
				cfg, err = img.ConfigFile()
				if err != nil {
					return nil, err
				}
				cfg.Architecture = p.Architecture
				cfg.OS = p.OS
				cfg.Variant = p.Variant
				cfg.Config.User = "65532:65532"
			} else {
				ref, err := name.ParseReference(base, name.WeakValidation)
				if err != nil {
					return nil, err
				}
				if tag, ok := ref.(name.Tag); ok && !opts.Pull {
					if !basesCached {
						progress("check local base %s for %s", tag.Name(), p.String())
					}
					img, err = st.Image(ctx, tag, p)
					if err == nil {
						if !basesCached {
							progress("found local base %s for %s", tag.Name(), p.String())
						}
					} else {
						if !basesCached {
							progress("base %s not found locally for %s", tag.Name(), p.String())
						}
					}
				}
				if img == nil {
					progress("resolve remote base %s for %s", ref.Name(), p.String())
					img, err = remote.Image(ref, remote.WithContext(ctx), remote.WithPlatform(p), remote.WithAuthFromKeychain(authn.DefaultKeychain))
					if err == nil {
						progress("resolved remote base %s for %s", ref.Name(), p.String())
					}
				}
				if err != nil {
					return nil, err
				}
				cfg, err = img.ConfigFile()
				if err != nil {
					return nil, err
				}
				if needsResolver {
					resolver, err = resolverFromImage(img)
					if err != nil {
						return nil, err
					}
				}
				cfg.Architecture = p.Architecture
				cfg.OS = p.OS
				cfg.Variant = p.Variant
			}
		case "ENV":
			pairs := parsePairs(in.Tokens, args)
			for k, v := range pairs {
				prefix := k + "="
				if i := slices.IndexFunc(cfg.Config.Env, func(e string) bool { return strings.HasPrefix(e, prefix) }); i >= 0 {
					cfg.Config.Env[i] = prefix + v
				} else {
					cfg.Config.Env = append(cfg.Config.Env, prefix+v)
				}
				args[k] = v
			}
		case "LABEL":
			pairs := parsePairs(in.Tokens, args)
			if cfg.Config.Labels == nil {
				cfg.Config.Labels = map[string]string{}
			}
			for k, v := range pairs {
				cfg.Config.Labels[k] = v
			}
		case "WORKDIR":
			cfg.Config.WorkingDir = cleanAbs(cfg.Config.WorkingDir, expand(in.Args, args))
		case "ENTRYPOINT":
			cfg.Config.Entrypoint = parseCommand(in)
		case "CMD":
			cfg.Config.Cmd = parseCommand(in)
		case "COPY":
			progress("create layer for COPY at line %d", in.Line)
			layer, err := copyLayer(ctxDir, in, args, cfg.Config.WorkingDir, matcher, resolver)
			if err != nil {
				return nil, err
			}
			img, err = mutate.AppendLayers(img, layer)
			if err != nil {
				return nil, err
			}
		}
	}
	if len(opts.Labels) > 0 {
		if cfg.Config.Labels == nil {
			cfg.Config.Labels = map[string]string{}
		}
		for k, v := range opts.Labels {
			cfg.Config.Labels[k] = v
		}
	}
	current, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}
	current.Architecture = cfg.Architecture
	current.OS = cfg.OS
	current.Variant = cfg.Variant
	current.Config = cfg.Config
	img, err = mutate.ConfigFile(img, current)
	if err != nil {
		return nil, err
	}
	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.ConfigMediaType(img, types.OCIConfigJSON)
	return img, nil
}

// attachSBOM generates a Syft SPDX JSON SBOM and stores it as an OCI referrer artifact.
func attachSBOM(ctx context.Context, st store.Store, tag name.Tag) error {
	if _, err := exec.LookPath("syft"); err != nil {
		return fmt.Errorf("--sbom requires syft on PATH; install syft or use --sbom=false")
	}
	repo := st.RepoPath(tag)
	cmd := exec.CommandContext(ctx, "syft", "oci-dir:"+repo+":"+tag.TagStr(), "-o", "spdx-json")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("syft failed to generate SBOM for %s: %s", tag.Name(), msg)
	}
	subject, err := st.Descriptor(ctx, tag)
	if err != nil {
		return err
	}
	artifact, err := sbomArtifact(out)
	if err != nil {
		return err
	}
	return st.PutArtifact(ctx, tag, subject, artifact, "sbom.spdx.json")
}

// sbomArtifact returns an OCI artifact image containing an SPDX JSON SBOM.
func sbomArtifact(doc []byte) (v1.Image, error) {
	layer := static.NewLayer(doc, types.MediaType("application/spdx+json"))
	img, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		return nil, err
	}
	return mutate.ConfigMediaType(mutate.MediaType(img, types.OCIManifestSchema1), types.MediaType("application/spdx+json")), nil
}

// defaultFile returns the default Dockerfile or Containerfile path for a context.
func defaultFile(ctxDir string) (string, error) {
	for _, n := range []string{"Dockerfile", "Containerfile"} {
		p := filepath.Join(ctxDir, n)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no Dockerfile or Containerfile found in %s", ctxDir)
}

// parsePlatforms parses Docker-style platform strings.
func parsePlatforms(in []string) ([]v1.Platform, error) {
	if len(in) == 0 {
		return []v1.Platform{{OS: runtime.GOOS, Architecture: runtime.GOARCH}}, nil
	}
	var out []v1.Platform
	for _, s := range in {
		for _, part := range strings.Split(s, ",") {
			p, err := v1.ParsePlatform(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			out = append(out, *p)
		}
	}
	return out, nil
}

// platformArgs returns automatic build arguments for a target platform.
func platformArgs(p v1.Platform) map[string]string {
	tp := p.OS + "/" + p.Architecture
	if p.Variant != "" {
		tp += "/" + p.Variant
	}
	return map[string]string{"TARGETPLATFORM": tp, "TARGETOS": p.OS, "TARGETARCH": p.Architecture, "TARGETVARIANT": p.Variant, "BUILDPLATFORM": runtime.GOOS + "/" + runtime.GOARCH, "BUILDOS": runtime.GOOS, "BUILDARCH": runtime.GOARCH}
}

// expand applies Dockerfile-style variable expansion for supported fields.
func expand(s string, vals map[string]string) string {
	return os.Expand(s, func(k string) string { return vals[k] })
}

// cleanAbs resolves p against the current working directory as an absolute container path.
func cleanAbs(wd, p string) string {
	if strings.HasPrefix(p, "/") {
		return path.Clean(p)
	}
	return path.Clean("/" + path.Join(wd, p))
}

// parsePairs converts parser tokens into key/value pairs.
func parsePairs(tokens []string, vals map[string]string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(tokens); {
		if i+2 < len(tokens) && tokens[i+2] == "=" {
			out[tokens[i]] = expand(unquote(tokens[i+1]), vals)
			i += 3
			continue
		}
		if k, v, ok := strings.Cut(tokens[i], "="); ok {
			out[k] = expand(unquote(v), vals)
			i++
			continue
		}
		if i+1 < len(tokens) {
			out[tokens[i]] = expand(unquote(tokens[i+1]), vals)
			i += 2
			continue
		}
		i++
	}
	return out
}

// parseCommand converts CMD or ENTRYPOINT into image config form.
func parseCommand(in containerfile.Instruction) []string {
	if in.JSON {
		return append([]string(nil), in.Tokens...)
	}
	return []string{"/bin/sh", "-c", in.Args}
}

// unquote removes parser-preserved quotes when present.
func unquote(s string) string {
	if unquoted, err := strconv.Unquote(s); err == nil {
		return unquoted
	}
	return s
}

// expandTokens expands variables in parser tokens.
func expandTokens(tokens []string, vals map[string]string) []string {
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, expand(unquote(token), vals))
	}
	return out
}

// contextPath resolves a COPY source and rejects paths outside the build context.
func contextPath(root, src string) (string, error) {
	cleaned := filepath.Clean(strings.TrimLeft(src, `/\`))
	if !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("COPY source %q is outside the build context", src)
	}
	return filepath.Join(root, cleaned), nil
}

// needsNamedChown reports whether any COPY uses a non-numeric chown value.
func needsNamedChown(cf *containerfile.File) bool {
	for _, in := range cf.Instructions {
		if in.Op != "COPY" {
			continue
		}
		for _, flag := range in.Flags {
			flag = strings.TrimPrefix(flag, "--")
			key, value, ok := strings.Cut(flag, "=")
			if !ok || key != "chown" {
				continue
			}
			owner, group, hasGroup := strings.Cut(value, ":")
			if _, err := strconv.Atoi(owner); err != nil {
				return true
			}
			if hasGroup && group != "" {
				if _, err := strconv.Atoi(group); err != nil {
					return true
				}
			}
		}
	}
	return false
}

// copyLayer creates one OCI layer for a COPY instruction.
func copyLayer(ctxDir string, in containerfile.Instruction, args map[string]string, workdir string, matcher *patternmatcher.PatternMatcher, resolver idResolver) (v1.Layer, error) {
	parts := expandTokens(in.Tokens, args)
	if len(parts) < 2 {
		return nil, fmt.Errorf("COPY at line %d requires source and destination", in.Line)
	}
	uid, gid, mode, err := copyMetadata(in.Flags, resolver)
	if err != nil {
		return nil, fmt.Errorf("COPY at line %d: %w", in.Line, err)
	}
	dest := parts[len(parts)-1]
	intoDir := strings.HasSuffix(dest, "/")
	if len(parts) > 2 && !intoDir {
		return nil, fmt.Errorf("COPY at line %d: destination must end with / when copying multiple sources", in.Line)
	}
	if !strings.HasPrefix(dest, "/") {
		dest = cleanAbs(workdir, dest)
	}
	sources := parts[:len(parts)-1]
	var files []copyEntry
	for _, src := range sources {
		srcPath, err := contextPath(ctxDir, src)
		if err != nil {
			return nil, fmt.Errorf("COPY at line %d: %w", in.Line, err)
		}
		matches, err := filepath.Glob(srcPath)
		if err != nil {
			return nil, fmt.Errorf("COPY at line %d: invalid source pattern %q", in.Line, src)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("COPY at line %d: source %q not found in build context", in.Line, src)
		}
		before := len(files)
		for _, m := range matches {
			if err := collect(ctxDir, m, dest, len(sources) > 1, intoDir, matcher, uid, gid, mode, &files); err != nil {
				return nil, err
			}
		}
		if len(files) == before {
			return nil, fmt.Errorf("COPY at line %d: source %q is excluded by .dockerignore", in.Line, src)
		}
	}
	files = addParentDirs(files)
	slices.SortFunc(files, func(a, b copyEntry) int { return strings.Compare(a.dst, b.dst) })
	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		pr, pw := io.Pipe()
		go func() {
			gw := gzip.NewWriter(pw)
			tw := tar.NewWriter(gw)
			var err error
			for _, e := range files {
				if err = writeEntry(tw, e); err != nil {
					break
				}
			}
			if cerr := tw.Close(); err == nil {
				err = cerr
			}
			if cerr := gw.Close(); err == nil {
				err = cerr
			}
			_ = pw.CloseWithError(err)
		}()
		return pr, nil
	}, tarball.WithMediaType(types.OCILayer))
}

type copyEntry struct {
	src, dst string
	info     os.FileInfo
	dir      bool
	uid      *int
	gid      *int
	mode     *int64
}

// addParentDirs synthesizes parent directory entries required by copied paths.
func addParentDirs(entries []copyEntry) []copyEntry {
	seen := map[string]bool{}
	for _, entry := range entries {
		name := strings.TrimPrefix(entry.dst, "/")
		if entry.dir {
			seen[strings.TrimSuffix(name, "/")+"/"] = true
		} else {
			seen[name] = true
		}
	}
	var parents []copyEntry
	mode := int64(0o755)
	for _, entry := range entries {
		dir := path.Dir(strings.TrimPrefix(entry.dst, "/"))
		var stack []string
		for dir != "." && dir != "/" {
			name := strings.TrimSuffix(dir, "/") + "/"
			if !seen[name] {
				stack = append(stack, name)
				seen[name] = true
			}
			dir = path.Dir(dir)
		}
		for i := len(stack) - 1; i >= 0; i-- {
			parents = append(parents, copyEntry{dst: stack[i], dir: true, mode: &mode})
		}
	}
	return append(entries, parents...)
}

// dockerignoreMatcher loads the Docker-compatible ignore matcher for a build.
func dockerignoreMatcher(root, dockerfile string) (*patternmatcher.PatternMatcher, error) {
	path := filepath.Join(root, ".dockerignore")
	if rel, err := filepath.Rel(root, dockerfile); err == nil {
		candidate := filepath.Join(root, rel+".dockerignore")
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
		}
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return patternmatcher.New(nil)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	patterns, err := ignorefile.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return patternmatcher.New(patterns)
}

// collect resolves source files into tar entries for a COPY instruction.
func collect(root, src, dest string, multiSource, destTrailingSlash bool, matcher *patternmatcher.PatternMatcher, uid, gid *int, mode *int64, out *[]copyEntry) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, src)
	rel = filepath.ToSlash(rel)
	ignored, _ := matcher.MatchesOrParentMatches(rel)
	if ignored && (!info.IsDir() || !matcher.Exclusions()) {
		return nil
	}
	baseDest := dest
	if !info.IsDir() && (multiSource || destTrailingSlash) {
		baseDest = path.Join(dest, path.Base(rel))
	} else if info.IsDir() && multiSource {
		baseDest = path.Join(dest, path.Base(rel))
	}
	if !info.IsDir() {
		*out = append(*out, copyEntry{src: src, dst: baseDest, info: info, uid: uid, gid: gid, mode: mode})
		return nil
	}
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rroot, _ := filepath.Rel(root, p)
		ignored, _ := matcher.MatchesOrParentMatches(filepath.ToSlash(rroot))
		if ignored {
			if info.IsDir() && !matcher.Exclusions() {
				return filepath.SkipDir
			}
			return nil
		}
		r, _ := filepath.Rel(src, p)
		if r == "." {
			return nil
		}
		entry := copyEntry{src: p, dst: path.Join(baseDest, filepath.ToSlash(r)), info: info, uid: uid, gid: gid, mode: mode}
		if info.IsDir() {
			entry.dir = true
		}
		*out = append(*out, entry)
		return nil
	})
}

// writeEntry writes one copy entry into a layer tar stream.
func writeEntry(tw *tar.Writer, e copyEntry) error {
	var hdr *tar.Header
	if e.info == nil {
		hdr = &tar.Header{Typeflag: tar.TypeDir, Mode: 0o755}
	} else {
		var err error
		hdr, err = tar.FileInfoHeader(e.info, "")
		if err != nil {
			return err
		}
	}
	hdr.Name = strings.TrimPrefix(e.dst, "/")
	if e.dir {
		hdr.Name = strings.TrimSuffix(hdr.Name, "/") + "/"
	}
	hdr.ModTime = hdr.ModTime.UTC()
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Uname = ""
	hdr.Gname = ""
	if e.uid != nil {
		hdr.Uid = *e.uid
	}
	if e.gid != nil {
		hdr.Gid = *e.gid
	}
	if e.mode != nil {
		hdr.Mode = *e.mode
	}
	if e.info != nil && e.info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(e.src)
		if err != nil {
			return err
		}
		hdr.Linkname = link
		return tw.WriteHeader(hdr)
	}
	if e.dir {
		return tw.WriteHeader(hdr)
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	f, err := os.Open(e.src)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(tw, f)
	return err
}

// copyMetadata parses COPY metadata flags such as chmod and chown.
func copyMetadata(flags []string, resolver idResolver) (*int, *int, *int64, error) {
	var uid *int
	var gid *int
	var mode *int64
	for _, flag := range flags {
		flag = strings.TrimPrefix(flag, "--")
		key, value, ok := strings.Cut(flag, "=")
		if !ok {
			return nil, nil, nil, fmt.Errorf("unsupported COPY flag %q", flag)
		}
		switch key {
		case "chmod":
			parsed, err := strconv.ParseInt(value, 8, 64)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("invalid --chmod value %q", value)
			}
			mode = &parsed
		case "chown":
			parsedUID, parsedGID, err := resolver.resolveChown(value)
			if err != nil {
				return nil, nil, nil, err
			}
			uid = &parsedUID
			gid = &parsedGID
		case "from":
			return nil, nil, nil, fmt.Errorf("COPY --from is not supported")
		default:
			return nil, nil, nil, fmt.Errorf("unsupported COPY flag --%s", key)
		}
	}
	return uid, gid, mode, nil
}
