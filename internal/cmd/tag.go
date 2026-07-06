// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/podplane/ocimage/pkg/store"
	"github.com/spf13/cobra"
)

// newTagCmd constructs the tag subcommand.
func newTagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tag SOURCE_IMAGE TARGET_IMAGE",
		Short: "Create a new tag in the ocimage OCI store",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveStoreRoot()
			if err != nil {
				return err
			}
			src, err := name.NewTag(args[0], name.WeakValidation)
			if err != nil {
				return err
			}
			dst, err := name.NewTag(args[1], name.WeakValidation)
			if err != nil {
				return err
			}
			if err := (store.Store{Root: root}).Tag(cmd.Context(), src, dst); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "tagged %s as %s\n", src.Name(), dst.Name())
			return err
		},
	}
}
