// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveStoreRoot resolves the CLI-managed OCI store root.
func resolveStoreRoot() (string, error) {
	if v := os.Getenv("OCIMAGE_STORE"); v != "" {
		return filepath.Abs(v)
	}
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "ocimage", "store"), nil
	}
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return filepath.Join(h, ".local", "state", "ocimage", "store"), nil
	}
	return filepath.Abs(filepath.Join(".ocimage", "store"))
}

// parseKeyValues parses repeated KEY=VALUE CLI flags into a map.
func parseKeyValues(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, v := range values {
		k, val, ok := strings.Cut(v, "=")
		if !ok {
			return nil, fmt.Errorf("expected KEY=VALUE, got %q", v)
		}
		out[k] = val
	}
	return out, nil
}
