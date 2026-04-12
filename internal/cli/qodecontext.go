package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/nqode/qode/internal/qodecontext"
	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage named work contexts",
	}
	cmd.AddCommand(
		newContextInitCmd(),
		newContextSwitchCmd(),
		newContextClearCmd(),
		newContextRemoveCmd(),
		newContextResetCmd(),
	)
	return cmd
}

func newContextInitCmd() *cobra.Command {
	var autoSwitch bool
	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Create a new context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextInit(cmd.Context(), cmd.OutOrStdout(), args[0], autoSwitch)
		},
	}
	cmd.Flags().BoolVar(&autoSwitch, "auto-switch", false, "switch to the new context after creation")
	return cmd
}

func runContextInit(ctx context.Context, out io.Writer, name string, autoSwitch bool) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	if err := qodecontext.Init(ctx, root, name); err != nil {
		return err
	}
	if autoSwitch {
		if err := qodecontext.Switch(ctx, root, name); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Created and switched to context: %s\n", name)
		return nil
	}
	_, _ = fmt.Fprintf(out, "Created context: %s\n", name)
	return nil
}

func newContextSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextSwitch(cmd.Context(), cmd.OutOrStdout(), args[0])
		},
	}
}

func runContextSwitch(ctx context.Context, out io.Writer, name string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	if err := qodecontext.Switch(ctx, root, name); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Switched to context: %s\n", name)
	return nil
}

func newContextClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear [name]",
		Short: "Clear a context's files, reinitialising stub files",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runContextClear(cmd.Context(), cmd.OutOrStdout(), name)
		},
	}
}

func runContextClear(ctx context.Context, out io.Writer, name string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	if name == "" {
		if name, err = qodecontext.CurrentName(ctx, root); err != nil {
			return err
		}
	}
	if err := qodecontext.Clear(ctx, root, name); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Cleared context: %s\n", name)
	return nil
}

func newContextRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove a context directory",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runContextRemove(cmd.Context(), cmd.OutOrStdout(), name)
		},
	}
}

func runContextRemove(ctx context.Context, out io.Writer, name string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	if name == "" {
		if name, err = qodecontext.CurrentName(ctx, root); err != nil {
			return err
		}
	}
	if err := qodecontext.Remove(ctx, root, name); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Removed context: %s\n", name)
	return nil
}

func newContextResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Clear the active context selection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContextReset(cmd.Context(), cmd.OutOrStdout())
		},
	}
}

func runContextReset(ctx context.Context, out io.Writer) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	if err := qodecontext.Reset(ctx, root); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, "Active context cleared.")
	return nil
}
