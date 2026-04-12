// Package cli defines the Cobra command tree and orchestrates the qode workflow.
package cli

import (
	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/version"
	"github.com/spf13/cobra"
)

var (
	rootCmd    *cobra.Command
	flagRoot   string
	flagStrict bool
)

// SetVersion sets the version string displayed by --version.
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd = &cobra.Command{
		Use:               "qode",
		Short:             "AI-assisted developer workflow CLI by nQode",
		PersistentPreRunE: checkVersion,
		Long: `qode is a general-purpose AI developer workflow tool by nQode.

It standardises how developers use AI coding assistants across client projects
with varied tech stacks — Next.js+React, .NET+React, Angular+Java, and more.

See 'qode workflow' for the full diagram.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(&flagRoot, "root", "", "project root directory (default: auto-detected)")
	rootCmd.PersistentFlags().BoolVar(&flagStrict, "strict", false, "enforce strict mode: gate violations cause non-zero exit")

	rootCmd.AddCommand(
		newInitCmd(),
		newPlanCmd(),
		newStartCmd(),
		newReviewCmd(),
		newContextCmd(),
		newKnowledgeCmd(),
		newWorkflowCmd(),
	)
}

// checkVersion is the PersistentPreRunE hook that enforces version compatibility
// between the running binary and the qode_version recorded in qode.yaml.
func checkVersion(cmd *cobra.Command, _ []string) error {
	// init is the remediation action — never block it.
	if cmd.Name() == "init" {
		return nil
	}
	// dev builds skip version checks entirely.
	if rootCmd.Version == "dev" || rootCmd.Version == "" {
		return nil
	}
	root, err := resolveRoot()
	if err != nil {
		return ErrNotInitialised
	}
	cfg, err := config.Load(root)
	if err != nil {
		return ErrNotInitialised
	}
	if cfg.QodeVersion == "" {
		return ErrNotInitialised
	}
	return version.CheckCompatibility(rootCmd.Version, cfg.QodeVersion)
}
