package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	rootCmd  *cobra.Command
	flagRoot string
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
		Use:   "qode",
		Short: "AI-assisted developer workflow CLI by nQode",
		Long: `qode is a general-purpose AI developer workflow tool by nQode.

It standardises how developers use AI coding assistants across client projects
with varied tech stacks — Next.js+React, .NET+React, Angular+Java, and more.

Workflow:
  1. qode branch create <name>                  # Create feature branch
  2. /qode-ticket-fetch <url>    (in IDE)       # Fetch ticket
  3. /qode-plan-refine           (in IDE)       # Refine requirements (3-5x → 25/25)
  4. /qode-plan-spec             (in IDE)       # Generate tech spec
  5. /qode-start                 (in IDE)       # Run implementation prompt
  6. /qode-review-code           (in IDE)       # Code review
  7. /qode-review-security       (in IDE)       # Security review
  8. /qode-knowledge-add-context (in IDE)       # Capture lessons learned
  9. qode check                                 # Run all quality gates
 10. qode branch remove <name>                  # Cleanup

See 'qode workflow' for the full diagram.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(&flagRoot, "root", "", "project root directory (default: auto-detected)")

	rootCmd.AddCommand(
		newInitCmd(),
		newPlanCmd(),
		newStartCmd(),
		newReviewCmd(),
		newCheckCmd(),
		newBranchCmd(),
		newIDECmd(),
		newTicketCmd(),
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

// writePromptToFile atomically writes content to path, creating parent dirs as needed.
// On template render error the caller should return before calling this.
func writePromptToFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".qode-prompt-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
