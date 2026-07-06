// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1"
)

// idResolver resolves COPY --chown users and groups from base image identity files.
// It tracks the effective /etc/passwd and /etc/group contents after applying
// base image layers and relevant OCI whiteouts.
type idResolver struct {
	users  map[string]userEntry
	groups map[string]int
}

type userEntry struct {
	uid int
	gid int
}

// resolverFromImage builds a user/group resolver from base image passwd and group files.
func resolverFromImage(img v1.Image) (idResolver, error) {
	resolver := idResolver{users: map[string]userEntry{}, groups: map[string]int{}}
	layers, err := img.Layers()
	if err != nil {
		return resolver, err
	}
	for _, layer := range layers {
		rc, err := layer.Uncompressed()
		if err != nil {
			return resolver, err
		}
		err = resolver.readLayer(rc)
		closeErr := rc.Close()
		if err != nil {
			return resolver, err
		}
		if closeErr != nil {
			return resolver, closeErr
		}
	}
	return resolver, nil
}

// readLayer scans a base layer for passwd and group files.
func (r *idResolver) readLayer(reader io.Reader) error {
	tr := tar.NewReader(reader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := path.Clean(strings.TrimPrefix(hdr.Name, "/"))
		if r.applyWhiteout(name) {
			continue
		}
		switch name {
		case "etc/passwd":
			if err := r.readPasswd(tr); err != nil {
				return err
			}
		case "etc/group":
			if err := r.readGroup(tr); err != nil {
				return err
			}
		}
	}
}

// applyWhiteout applies OCI whiteout entries that affect passwd or group data.
func (r *idResolver) applyWhiteout(name string) bool {
	switch name {
	case ".wh.etc", "etc/.wh..wh..opq":
		r.users = map[string]userEntry{}
		r.groups = map[string]int{}
		return true
	case "etc/.wh.passwd":
		r.users = map[string]userEntry{}
		return true
	case "etc/.wh.group":
		r.groups = map[string]int{}
		return true
	default:
		return strings.Contains(path.Base(name), ".wh.")
	}
}

// readPasswd records users from an /etc/passwd stream.
func (r *idResolver) readPasswd(reader io.Reader) error {
	users := map[string]userEntry{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 4 || fields[0] == "" {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		gid, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}
		users[fields[0]] = userEntry{uid: uid, gid: gid}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	r.users = users
	return nil
}

// readGroup records groups from an /etc/group stream.
func (r *idResolver) readGroup(reader io.Reader) error {
	groups := map[string]int{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 3 || fields[0] == "" {
			continue
		}
		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		groups[fields[0]] = gid
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	r.groups = groups
	return nil
}

// resolveChown resolves a Docker-style chown value into uid and gid.
func (r idResolver) resolveChown(value string) (int, int, error) {
	owner, group, hasGroup := strings.Cut(value, ":")
	uid, primaryGID, err := r.resolveUser(owner)
	if err != nil {
		return 0, 0, err
	}
	if !hasGroup || group == "" {
		return uid, primaryGID, nil
	}
	gid, err := r.resolveGroup(group)
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
}

// resolveUser resolves a numeric or named user.
func (r idResolver) resolveUser(owner string) (int, int, error) {
	uid, err := strconv.Atoi(owner)
	if err == nil {
		return uid, uid, nil
	}
	user, ok := r.users[owner]
	if !ok {
		return 0, 0, fmt.Errorf("unable to resolve user %q for COPY --chown", owner)
	}
	return user.uid, user.gid, nil
}

// resolveGroup resolves a numeric or named group.
func (r idResolver) resolveGroup(group string) (int, error) {
	gid, err := strconv.Atoi(group)
	if err == nil {
		return gid, nil
	}
	gid, ok := r.groups[group]
	if !ok {
		return 0, fmt.Errorf("unable to resolve group %q for COPY --chown", group)
	}
	return gid, nil
}
