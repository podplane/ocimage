// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/podplane/ocimage/pkg/initfile"
	"github.com/spf13/cobra"
)

type initFlags struct {
	template string
	force    bool
	list     bool
}

// newInitCmd constructs the init subcommand.
func newInitCmd() *cobra.Command {
	var flags initFlags
	cmd := &cobra.Command{
		Use:   "init [OPTIONS] [PATH]",
		Short: "Generate a Containerfile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if flags.list {
				for _, tmpl := range initfile.Templates() {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", tmpl.ID, tmpl.Description); err != nil {
						return err
					}
				}
				return nil
			}

			tmpl, err := resolveInitTemplate(dir, flags.template)
			if err != nil {
				return err
			}
			written, err := initfile.Write(dir, tmpl, flags.force)
			if err != nil {
				return err
			}
			for _, path := range written {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "created %s from %s\n", path, tmpl.ID); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flags.template, "template", "", "template ID to generate")
	cmd.Flags().BoolVar(&flags.force, "force", false, "overwrite generated files if they already exist")
	cmd.Flags().BoolVar(&flags.list, "list", false, "list available template IDs")
	return cmd
}

// resolveInitTemplate returns the explicitly requested template or the single detected template.
func resolveInitTemplate(dir string, id string) (initfile.Template, error) {
	if id != "" {
		tmpl, ok := initfile.FindTemplate(id)
		if !ok {
			return initfile.Template{}, fmt.Errorf("unknown template %q; run ocimage init --list", id)
		}
		return tmpl, nil
	}
	matches, err := initfile.Detect(dir)
	if err != nil {
		return initfile.Template{}, err
	}
	if len(matches) == 1 {
		return matches[0].Template, nil
	}
	if len(matches) == 0 {
		return initfile.Template{}, fmt.Errorf("could not detect a supported project; pass --template (run ocimage init --list)")
	}

	var b strings.Builder
	b.WriteString("detected multiple templates; pass --template with one of:\n")
	for _, match := range matches {
		_, _ = fmt.Fprintf(&b, "  %s\t%s\n", match.Template.ID, match.Reason)
	}
	return initfile.Template{}, fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}
