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

// newPushCmd constructs the push subcommand.
func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push NAME[:TAG]",
		Short: "Push an image from the ocimage OCI store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveStoreRoot()
			if err != nil {
				return err
			}
			ref, err := name.NewTag(args[0], name.WeakValidation)
			if err != nil {
				return err
			}
			if err := (store.Store{Root: root}).Push(cmd.Context(), ref); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "pushed %s\n", ref.Name())
			return err
		},
	}
}
