// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package build

import "github.com/google/go-containerregistry/pkg/v1"

type Options struct {
	ContextDir string
	File       string
	Tags       []string
	Platforms  []string
	BuildArgs  map[string]string
	Labels     map[string]string
	StoreRoot  string
	Push       bool
	Pull       bool
	SBOM       bool
	Docker     string
	Progress   func(string)
}

type Result struct {
	Tags      []string
	Platforms []v1.Platform
	Pushed    []string
	SBOMs     []string
}
