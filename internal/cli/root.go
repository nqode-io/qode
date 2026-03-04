package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd *cobra.Command
	// GlobalFlags shared across commands
	flagRoot    string
	flagVerbose bool
)

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd = &cobra.Command{
		Use:   "qode",
		Short: "AI-assisted developer workflow CLI for nQode projects",
		Long: `qode is a general-purpose AI developer workflow tool for nQode.

It standardises how developers use AI coding assistants across client projects
with varied tech stacks — Next.js+React, .NET+React, Angular+Java, and more.

Workflow:
  1. qode branch create <name>            # Create feature branch
  2. qode ticket fetch <url>              # Fetch ticket (or /qode-ticket-fetch in IDE)
  3. /qode-plan-refine  (in IDE)          # Refine requirements (3-5x → 25/25)
  4. /qode-plan-spec    (in IDE)          # Generate tech spec
  5. qode start                           # Run implementation (or /qode-start in IDE)
  6. /qode-review-code  (in IDE)          # Code review
  7. /qode-review-security (in IDE)       # Security review
  8. qode check                           # Run all quality gates
  9. qode branch remove <name>            # Cleanup

See 'qode workflow' for the full diagram.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVar(&flagRoot, "root", "", "project root directory (default: auto-detected)")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(
		newInitCmd(),
		newPlanCmd(),
		newStartCmd(),
		newReviewCmd(),
		newCheckCmd(),
		newBranchCmd(),
		newIDECmd(),
		newTicketCmd(),
		newConfigCmd(),
		newKnowledgeCmd(),
		newHelpWorkflowCmd(),
	)
}

// resolveRoot returns the effective project root, preferring the --root flag,
// then the current working directory.
func resolveRoot() (string, error) {
	if flagRoot != "" {
		return flagRoot, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	return wd, nil
}
