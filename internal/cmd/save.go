// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/podplane/ocimage/pkg/store"
	"github.com/spf13/cobra"
)

type saveFlags struct {
	output   string
	platform string
}

// newSaveCmd constructs the save subcommand.
func newSaveCmd() *cobra.Command {
	var flags saveFlags
	cmd := &cobra.Command{
		Use:   "save [OPTIONS] IMAGE [IMAGE...]",
		Short: "Save images from the ocimage OCI store to a Docker-compatible tar archive",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveStoreRoot()
			if err != nil {
				return err
			}
			refs := make([]name.Tag, 0, len(args))
			for _, arg := range args {
				ref, err := name.NewTag(arg, name.WeakValidation)
				if err != nil {
					return err
				}
				refs = append(refs, ref)
			}
			var platform *v1.Platform
			if flags.platform != "" {
				platform, err = v1.ParsePlatform(flags.platform)
				if err != nil {
					return fmt.Errorf("invalid --platform %q: %w", flags.platform, err)
				}
			}
			if flags.output == "" {
				return (store.Store{Root: root}).Save(cmd.Context(), refs, platform, cmd.OutOrStdout())
			}
			f, err := os.Create(flags.output)
			if err != nil {
				return err
			}
			err = (store.Store{Root: root}).Save(cmd.Context(), refs, platform, f)
			closeErr := f.Close()
			if err != nil {
				return err
			}
			return closeErr
		},
	}
	cmd.Flags().StringVarP(&flags.output, "output", "o", "", "write to a file, instead of STDOUT")
	cmd.Flags().StringVar(&flags.platform, "platform", "", "save only the given platform from a multi-platform image")
	return cmd
}
