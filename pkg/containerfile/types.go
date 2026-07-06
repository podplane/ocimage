// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package containerfile

type File struct{ Instructions []Instruction }

type Instruction struct {
	Op       string
	Args     string
	Flags    []string
	Tokens   []string
	JSON     bool
	Line     int
	Original string
}
