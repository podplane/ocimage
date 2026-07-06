// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package buildvars

var (
	buildVersion = "dev"
	buildDate    = "unknown"
	commitHash   = "unknown"
	commitDate   = "unknown"
	commitBranch = "unknown"
)

// Version returns the linker-injected build version.
func Version() string { return buildVersion }

// BuildDate returns the linker-injected build timestamp.
func BuildDate() string { return buildDate }

// CommitHash returns the linker-injected source commit hash.
func CommitHash() string { return commitHash }

// CommitDate returns the linker-injected source commit timestamp.
func CommitDate() string { return commitDate }

// CommitBranch returns the linker-injected source branch name.
func CommitBranch() string { return commitBranch }
