// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/podplane/ocimage/internal/buildvars"
	"github.com/spf13/cobra"
)

// NewRootCmd constructs the root ocimage command.
func NewRootCmd() *cobra.Command {
	cobra.EnableCommandSorting = false
	cmd := &cobra.Command{
		Use:     "ocimage",
		Short:   "Build and push lightweight OCI images",
		Version: buildvars.Version(),
	}
	cmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
	cmd.AddCommand(newBuildCmd(), newInitCmd(), newPushCmd(), newSaveCmd(), newTagCmd(), newVersionCmd())
	return cmd
}
