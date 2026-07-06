// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"archive/tar"
	"bytes"
	"testing"
)

// TestChownResolverSupportsNames verifies named user and group resolution.
func TestChownResolverSupportsNames(t *testing.T) {
	resolver := idResolver{users: map[string]userEntry{"app": {uid: 1000, gid: 1001}}, groups: map[string]int{"apps": 2000}}
	uid, gid, mode, err := copyMetadata([]string{"--chown=app:apps", "--chmod=755"}, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if uid == nil || *uid != 1000 || gid == nil || *gid != 2000 || mode == nil || *mode != 0o755 {
		t.Fatalf("metadata uid=%v gid=%v mode=%v", uid, gid, mode)
	}
}

// TestChownResolverRejectsUnknownNames verifies unknown named users are rejected.
func TestChownResolverRejectsUnknownNames(t *testing.T) {
	_, _, _, err := copyMetadata([]string{"--chown=app"}, idResolver{})
	if err == nil {
		t.Fatal("expected unknown user error")
	}
}

// TestChownResolverHonorsWhiteouts verifies passwd and group whiteouts affect name resolution.
func TestChownResolverHonorsWhiteouts(t *testing.T) {
	resolver := idResolver{users: map[string]userEntry{}, groups: map[string]int{}}
	if err := resolver.readLayer(testTar(t, map[string]string{
		"etc/passwd": "app:x:1000:1000::/home/app:/bin/sh\n",
		"etc/group":  "app:x:1000:\n",
	})); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolver.resolveChown("app:app"); err != nil {
		t.Fatal(err)
	}
	if err := resolver.readLayer(testTar(t, map[string]string{"etc/.wh.passwd": ""})); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolver.resolveChown("app:app"); err == nil {
		t.Fatal("expected passwd whiteout to remove app user")
	}
	if err := resolver.readLayer(testTar(t, map[string]string{
		"etc/passwd": "app:x:2000:2000::/home/app:/bin/sh\n",
		"etc/group":  "app:x:2000:\n",
	})); err != nil {
		t.Fatal(err)
	}
	uid, gid, err := resolver.resolveChown("app:app")
	if err != nil {
		t.Fatal(err)
	}
	if uid != 2000 || gid != 2000 {
		t.Fatalf("resolved uid=%d gid=%d, want 2000:2000", uid, gid)
	}
	if err := resolver.readLayer(testTar(t, map[string]string{"etc/.wh..wh..opq": ""})); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolver.resolveChown("app:app"); err == nil {
		t.Fatal("expected opaque etc whiteout to remove app user")
	}
}

// testTar returns an in-memory tar stream containing files by path.
func testTar(t *testing.T, files map[string]string) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}
