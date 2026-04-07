package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	gocontext "github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/knowledge"
	"github.com/nqode/qode/internal/prompt"
	"github.com/spf13/cobra"
)

func newKnowledgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "knowledge",
		Short: "Manage the project knowledge base",
	}
	cmd.AddCommand(newKnowledgeListCmd(), newKnowledgeAddCmd(), newKnowledgeSearchCmd(), newKnowledgeAddBranchCmd())
	return cmd
}

func newKnowledgeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List knowledge base files",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			files, err := knowledge.List(root, cfg)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				fmt.Println("Knowledge base is empty.")
				fmt.Printf("Add files with: qode knowledge add <path>\n")
				return nil
			}
			for _, f := range files {
				fmt.Println(f)
			}
			return nil
		},
	}
}

func newKnowledgeAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a file to the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			src := args[0]

			kbDir := filepath.Join(root, config.QodeDir, "knowledge")
			if err := os.MkdirAll(kbDir, 0755); err != nil {
				return err
			}

			data, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("reading %s: %w", src, err)
			}

			dest := filepath.Join(kbDir, filepath.Base(src))
			if err := os.WriteFile(dest, data, 0644); err != nil {
				return err
			}
			fmt.Printf("Added to knowledge base: %s\n", dest)
			return nil
		},
	}
}

func newKnowledgeSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(root)
			if err != nil {
				return err
			}
			results, err := knowledge.Search(root, cfg, args[0])
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Printf("No results for %q\n", args[0])
				return nil
			}
			for _, r := range results {
				fmt.Printf("%s: %s\n", r.File, r.Snippet)
			}
			return nil
		},
	}
}

const maxDiffLines = 500

func newKnowledgeAddBranchCmd() *cobra.Command {
	var toFile bool
	cmd := &cobra.Command{
		Use:   "add-branch [branches...]",
		Short: "Generate a lesson extraction prompt from branch context",
		Long:  "Reads branch artifacts (ticket, spec, reviews, diff) and writes a lesson extraction prompt to stdout.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKnowledgeAddBranch(args, toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func runKnowledgeAddBranch(args []string, toFile bool) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	branch, err := git.CurrentBranch(root)
	if err != nil {
		return err
	}
	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	branches := parseBranchArgs(args)
	fmt.Fprintf(os.Stderr, "Extracting lessons from branches: %s\n", strings.Join(branches, ", "))

	data, err := buildBranchLessonData(root, engine, branches, branch)
	if err != nil {
		return err
	}

	p, err := engine.Render("knowledge/add-branch", data)
	if err != nil {
		return err
	}

	if toFile {
		branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
		promptPath := filepath.Join(branchDir, ".knowledge-add-branch-prompt.md")
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Lesson extraction prompt saved to:\n  %s\n", promptPath)
		return nil
	}

	_, err = fmt.Print(p)
	return err
}

func buildBranchLessonData(root string, engine *prompt.Engine, branches []string, currentBranch string) (prompt.TemplateData, error) {
	var allTicket, allAnalysis, allSpec, allExtra strings.Builder
	var diff string

	for _, b := range branches {
		ctx, err := gocontext.Load(root, b)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: branch context %q: %v\n", b, err)
			continue
		}
		if ctx.Ticket != "" {
			allTicket.WriteString(ctx.Ticket)
			allTicket.WriteString("\n\n")
		}
		if ctx.RefinedAnalysis != "" {
			allAnalysis.WriteString(ctx.RefinedAnalysis)
			allAnalysis.WriteString("\n\n")
		}
		if ctx.Spec != "" {
			allSpec.WriteString(ctx.Spec)
			allSpec.WriteString("\n\n")
		}

		for _, reviewFile := range []string{"code-review.md", "security-review.md"} {
			data, readErr := os.ReadFile(filepath.Join(ctx.ContextDir, reviewFile))
			if readErr == nil && len(data) > 0 {
				allExtra.WriteString("### ")
				allExtra.WriteString(reviewFile)
				allExtra.WriteString("\n\n")
				allExtra.Write(data)
				allExtra.WriteString("\n\n")
			}
		}
	}

	d, err := git.DiffFromBase(root, "")
	if err == nil {
		diff = truncateLines(d, maxDiffLines)
	}

	lessons, _ := knowledge.ListLessons(root)
	lessonsStr := formatLessonsList(lessons)

	branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(currentBranch))

	return prompt.TemplateData{
		Project:   prompt.TemplateProject{Name: engine.ProjectName()},
		Branch:    currentBranch,
		Ticket:    allTicket.String(),
		Analysis:  allAnalysis.String(),
		Spec:      allSpec.String(),
		Diff:      diff,
		Extra:     allExtra.String(),
		Lessons:   lessonsStr,
		BranchDir: branchDir,
	}, nil
}

func parseBranchArgs(args []string) []string {
	var branches []string
	for _, arg := range args {
		for _, b := range strings.Split(arg, ",") {
			b = strings.TrimSpace(b)
			if b != "" && b != "." && !strings.Contains(b, "..") {
				branches = append(branches, b)
			}
		}
	}
	return branches
}

func formatLessonsList(lessons []knowledge.LessonSummary) string {
	if len(lessons) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, l := range lessons {
		fmt.Fprintf(&sb, "- **%s**: %s\n", l.Title, l.Summary)
	}
	return sb.String()
}

func truncateLines(s string, max int) string {
	lines := strings.SplitN(s, "\n", max+1)
	if len(lines) <= max {
		return s
	}
	return strings.Join(lines[:max], "\n") + "\n\n(truncated)"
}
