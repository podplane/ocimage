// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/podplane/ocimage/internal/buildvars"
	"github.com/spf13/cobra"
)

// newVersionCmd constructs the version subcommand.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{Use: "version", Short: "Print version", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s (%s)\n", buildvars.Version(), buildvars.CommitHash())
	}}
}
