package cli

import (
	"fmt"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/detect"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate qode configuration",
	}
	cmd.AddCommand(newConfigShowCmd(), newConfigDetectCmd(), newConfigValidateCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display the resolved configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Print(string(data))
			return nil
		},
	}
}

func newConfigDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Auto-detect tech stacks and show what qode would configure",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}

			layers, err := detect.Composite(root)
			if err != nil {
				return err
			}

			if len(layers) == 0 {
				fmt.Println("No recognised tech stacks detected.")
				return nil
			}

			fmt.Printf("%-20s  %-12s  %-8s  %s\n", "Path", "Stack", "Conf.", "Suggested name")
			fmt.Println("--------------------  ------------  --------  --------------")
			for _, l := range layers {
				fmt.Printf("%-20s  %-12s  %5.0f%%   %s\n",
					l.Path+"/", l.Stack, l.Confidence*100, l.Name)
			}
			return nil
		},
	}
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate qode.yaml for schema correctness",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return fmt.Errorf("invalid qode.yaml: %w", err)
			}

			var errs []string
			if cfg.Project.Name == "" {
				errs = append(errs, "project.name is required")
			}
			if len(cfg.Layers()) == 0 {
				errs = append(errs, "at least one layer (or project.stack) is required")
			}
			for _, l := range cfg.Project.Layers {
				if l.Stack == "" {
					errs = append(errs, fmt.Sprintf("layer %q: stack is required", l.Name))
				}
			}

			if len(errs) > 0 {
				fmt.Println("Validation errors:")
				for _, e := range errs {
					fmt.Printf("  - %s\n", e)
				}
				return fmt.Errorf("validation failed")
			}

			fmt.Println("qode.yaml is valid.")
			fmt.Printf("  Project:  %s\n", cfg.Project.Name)
			fmt.Printf("  Topology: %s\n", cfg.Project.Topology)
			fmt.Printf("  Layers:   %d\n", len(cfg.Layers()))
			return nil
		},
	}
}
