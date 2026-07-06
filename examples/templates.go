// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

// Package examples embeds ocimage example packaging templates.
package examples

import "embed"

// Files contains embedded files used by ocimage init.
//
//go:embed */*/Containerfile* typescript/spa-tanstack-start/Caddyfile
var Files embed.FS
