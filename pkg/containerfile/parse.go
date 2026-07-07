// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package containerfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// UnsupportedError describes Containerfile syntax outside ocimage's supported subset.
type UnsupportedError struct {
	Path    string
	Line    int
	Op      string
	Message string
	Hint    string
}

// Error formats e with file and line context.
func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("%s:%d: %s.\n\n%s", e.Path, e.Line, e.Message, e.Hint)
}

// ParseFile parses and validates a Containerfile or Dockerfile.
func ParseFile(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	res, err := parser.Parse(f)
	if err != nil {
		return nil, err
	}
	cf := &File{}
	for _, n := range res.AST.Children {
		op := strings.ToUpper(n.Value)
		args := strings.TrimSpace(n.Original)
		if idx := strings.Index(strings.ToUpper(args), op); idx >= 0 {
			args = strings.TrimSpace(args[idx+len(op):])
		}
		inst := Instruction{Op: op, Args: args, Flags: n.Flags, Tokens: nodeTokens(n.Next), JSON: n.Attributes["json"], Line: n.StartLine, Original: n.Original}
		cf.Instructions = append(cf.Instructions, inst)
	}
	return cf, Validate(path, cf)
}

// Validate ensures a parsed file uses only the supported instruction subset.
func Validate(path string, cf *File) error {
	seenFrom := false
	for _, in := range cf.Instructions {
		if !seenFrom && in.Op != "ARG" && in.Op != "FROM" {
			return validationError(path, in, in.Op+" cannot appear before FROM", "only ARG may appear before the first FROM instruction")
		}
		switch in.Op {
		case "FROM":
			if seenFrom {
				return validationError(path, in, "multi-stage builds are not supported", "ocimage packages prebuilt artifacts; build artifacts first, then COPY them into one final image")
			}
			seenFrom = true
			if len(in.Flags) > 0 {
				return validationError(path, in, "FROM flags are not supported", "use ocimage build --platform instead of FROM --platform")
			}
			parts := strings.Fields(in.Args)
			for i, p := range parts {
				if strings.EqualFold(p, "AS") || (i > 0 && strings.HasPrefix(strings.ToUpper(p), "AS")) {
					return validationError(path, in, "multi-stage builds are not supported", "remove the stage name and COPY prebuilt files from the build context")
				}
			}
		case "ARG", "COPY", "WORKDIR", "ENV", "LABEL", "ENTRYPOINT", "CMD":
			if in.Op == "COPY" {
				for _, f := range in.Flags {
					flag := strings.TrimPrefix(f, "--")
					key, _, ok := strings.Cut(flag, "=")
					if !ok {
						return validationError(path, in, "unsupported COPY flag "+f, "ocimage supports COPY --chmod and --chown; use Docker fallback for other COPY flags")
					}
					switch key {
					case "chmod", "chown":
						continue
					case "from":
						return validationError(path, in, "COPY --from is not supported", "ocimage does not support multi-stage builds")
					default:
						return validationError(path, in, "unsupported COPY flag --"+key, "ocimage supports COPY --chmod and --chown; use Docker fallback for other COPY flags")
					}
				}
			}
		case "ADD", "RUN", "USER", "EXPOSE", "VOLUME", "STOPSIGNAL", "SHELL", "ONBUILD", "HEALTHCHECK":
			return validationError(path, in, in.Op+" is not supported by ocimage", "ocimage creates OCI images from prebuilt artifacts and does not execute build steps or support this image metadata instruction")
		default:
			return validationError(path, in, in.Op+" is not supported by ocimage", "use the supported subset: FROM, ARG, COPY, WORKDIR, ENV, LABEL, ENTRYPOINT, CMD")
		}
	}
	if !seenFrom {
		return fmt.Errorf("%s: missing FROM instruction", path)
	}
	return nil
}

// validationError returns a source-positioned unsupported syntax error.
func validationError(path string, in Instruction, msg, hint string) error {
	return &UnsupportedError{Path: path, Line: in.Line, Op: in.Op, Message: msg, Hint: hint}
}

// nodeTokens flattens parser nodes into token values.
func nodeTokens(n *parser.Node) []string {
	var tokens []string
	for ; n != nil; n = n.Next {
		if n.Value != "" {
			tokens = append(tokens, n.Value)
		}
	}
	return tokens
}
