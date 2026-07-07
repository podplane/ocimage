// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strconv"

	buildlib "github.com/podplane/ocimage/pkg/build"
	"github.com/spf13/cobra"
)

type buildFlags struct {
	file                  string
	tags                  []string
	platform              string
	buildArgs             []string
	labels                []string
	push                  bool
	pull                  bool
	docker                string
	sbom                  string
	unsupportedTarget     string
	unsupportedNoCache    bool
	unsupportedProvenance string
	unsupportedOutput     []string
}

// newBuildCmd constructs the build subcommand.
func newBuildCmd() *cobra.Command {
	var flags buildFlags
	cmd := &cobra.Command{
		Use:   "build [OPTIONS] PATH",
		Short: "Build an OCI image from a Containerfile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.unsupportedTarget != "" {
				return fmt.Errorf("--target is not supported because ocimage does not support multi-stage builds")
			}
			if flags.unsupportedNoCache {
				return fmt.Errorf("--no-cache is not supported because ocimage does not execute build steps or maintain a build cache")
			}
			if flags.unsupportedProvenance != "" {
				return fmt.Errorf("--provenance is not supported yet")
			}
			sbom, err := strconv.ParseBool(flags.sbom)
			if err != nil {
				return fmt.Errorf("--sbom=%s is not supported; ocimage supports boolean values", flags.sbom)
			}
			if len(flags.unsupportedOutput) > 0 {
				return fmt.Errorf("--output is not supported yet; ocimage stores builds in the OCI store and can push them with ocimage push")
			}
			ctxDir := "."
			if len(args) == 1 {
				ctxDir = args[0]
			}
			storeRoot, err := resolveStoreRoot()
			if err != nil {
				return err
			}
			buildArgs, err := parseKeyValues(flags.buildArgs)
			if err != nil {
				return err
			}
			labels, err := parseKeyValues(flags.labels)
			if err != nil {
				return err
			}
			platforms := []string{}
			if flags.platform != "" {
				platforms = []string{flags.platform}
			}
			progressc := make(chan string, 64)
			progressDone := make(chan error, 1)
			go func() {
				var writeErr error
				for msg := range progressc {
					if writeErr != nil {
						continue
					}
					_, writeErr = fmt.Fprintf(cmd.OutOrStdout(), "=> %s\n", msg)
				}
				progressDone <- writeErr
			}()
			progress := func(msg string) {
				select {
				case progressc <- msg:
				case <-cmd.Context().Done():
				}
			}
			res, err := buildlib.Build(cmd.Context(), buildlib.Options{ContextDir: ctxDir, File: flags.file, Tags: flags.tags, Platforms: platforms, BuildArgs: buildArgs, Labels: labels, StoreRoot: storeRoot, Push: flags.push, Pull: flags.pull, SBOM: sbom, Docker: flags.docker, Progress: progress})
			close(progressc)
			if progressErr := <-progressDone; progressErr != nil && err == nil {
				return progressErr
			}
			if err != nil {
				return err
			}
			for _, tag := range res.Tags {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "built %s\n", tag); err != nil {
					return err
				}
			}
			for _, tag := range res.Pushed {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "pushed %s\n", tag); err != nil {
					return err
				}
			}
			for _, tag := range res.SBOMs {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "generated sbom for %s\n", tag); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&flags.file, "file", "f", "", "name of the Containerfile or Dockerfile")
	cmd.Flags().StringArrayVarP(&flags.tags, "tag", "t", nil, "name and optionally a tag")
	cmd.Flags().StringVar(&flags.platform, "platform", "", "target platform(s), comma-separated")
	cmd.Flags().StringArrayVar(&flags.buildArgs, "build-arg", nil, "set build-time variables")
	cmd.Flags().StringArrayVar(&flags.labels, "label", nil, "set image labels")
	cmd.Flags().BoolVar(&flags.push, "push", false, "push after building")
	cmd.Flags().BoolVar(&flags.pull, "pull", false, "always attempt to pull base images instead of using the local OCI store first")
	cmd.Flags().StringVar(&flags.docker, "docker", "", "use Docker Buildx fallback, optionally with a Docker binary path")
	cmd.Flags().Lookup("docker").NoOptDefVal = "docker"
	cmd.Flags().StringVar(&flags.sbom, "sbom", "false", "generate an SBOM attestation with syft")
	cmd.Flags().Lookup("sbom").NoOptDefVal = "true"
	cmd.Flags().StringVar(&flags.unsupportedTarget, "target", "", "unsupported")
	cmd.Flags().BoolVar(&flags.unsupportedNoCache, "no-cache", false, "unsupported")
	cmd.Flags().StringVar(&flags.unsupportedProvenance, "provenance", "", "unsupported")
	cmd.Flags().StringArrayVarP(&flags.unsupportedOutput, "output", "o", nil, "unsupported")
	return cmd
}
