package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/knowledge"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/qodecontext"
	"github.com/spf13/cobra"
)

func newKnowledgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "knowledge",
		Short: "Manage the project knowledge base",
	}
	cmd.AddCommand(newKnowledgeListCmd(), newKnowledgeAddCmd(), newKnowledgeSearchCmd(), newKnowledgeAddContextCmd())
	return cmd
}

func newKnowledgeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List knowledge base files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKnowledgeList(cmd.OutOrStdout())
		},
	}
}

func runKnowledgeList(out io.Writer) error {
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
		_, _ = fmt.Fprintln(out, "Knowledge base is empty.")
		_, _ = fmt.Fprintf(out, "Add files with: qode knowledge add <path>\n")
		return nil
	}
	for _, f := range files {
		_, _ = fmt.Fprintln(out, f)
	}
	return nil
}

func newKnowledgeAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a file to the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKnowledgeAdd(cmd.OutOrStdout(), args[0])
		},
	}
}

func runKnowledgeAdd(out io.Writer, src string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	kbDir := filepath.Join(root, config.QodeDir, "knowledge")
	if err := iokit.EnsureDir(kbDir); err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}

	dest := filepath.Join(kbDir, filepath.Base(src))
	if err := iokit.WriteFile(dest, data, 0644); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Added to knowledge base: %s\n", dest)
	return nil
}

func newKnowledgeSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKnowledgeSearch(cmd.OutOrStdout(), args[0])
		},
	}
}

func runKnowledgeSearch(out io.Writer, query string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	results, err := knowledge.Search(root, cfg, query)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintf(out, "No results for %q\n", query)
		return nil
	}
	for _, r := range results {
		_, _ = fmt.Fprintf(out, "%s: %s\n", r.File, r.Snippet)
	}
	return nil
}

const maxDiffLines = 500

func newKnowledgeAddContextCmd() *cobra.Command {
	var toFile bool
	cmd := &cobra.Command{
		Use:   "add-context",
		Short: "Generate a lesson extraction prompt from the current context",
		Long:  "Reads context artifacts (ticket, spec, reviews, diff) and writes a lesson extraction prompt to stdout.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKnowledgeAddContext(cmd.OutOrStdout(), cmd.ErrOrStderr(), toFile)
		},
	}
	cmd.Flags().BoolVar(&toFile, "to-file", false, "save prompt to file instead of stdout")
	return cmd
}

func runKnowledgeAddContext(out, errOut io.Writer, toFile bool) error {
	sess, err := loadSession()
	if err != nil {
		return err
	}

	data, err := buildContextLessonData(sess.Root, sess.Engine, sess.Context, sess.Config)
	if err != nil {
		return err
	}

	p, err := sess.Engine.Render("knowledge/add-context-artifacts", data)
	if err != nil {
		return err
	}

	if toFile {
		promptPath := filepath.Join(sess.Context.ContextDir, ".knowledge-add-context-prompt.md")
		if err := writePromptToFile(promptPath, p); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(errOut, "Lesson extraction prompt saved to:\n  %s\n", promptPath)
		return nil
	}

	_, err = fmt.Fprint(out, p)
	return err
}

func buildContextLessonData(root string, engine prompt.Renderer, ctx *qodecontext.Context, cfg *config.Config) (prompt.TemplateData, error) {
	var reviewExtra strings.Builder
	for _, reviewFile := range []string{"code-review.md", "security-review.md"} {
		data, readErr := os.ReadFile(filepath.Join(ctx.ContextDir, reviewFile))
		if readErr == nil && len(data) > 0 {
			reviewExtra.WriteString("### ")
			reviewExtra.WriteString(reviewFile)
			reviewExtra.WriteString("\n\n")
			reviewExtra.Write(data)
			reviewExtra.WriteString("\n\n")
		}
	}

	diff := truncateLines(runDiffCommand(root, cfg.Diff.Command), maxDiffLines)

	lessons, _ := knowledge.ListLessons(root)
	lessonsStr := formatLessonsList(lessons)

	return prompt.NewTemplateData(engine.ProjectName()).
		WithTicket(ctx.Ticket).
		WithAnalysis(ctx.RefinedAnalysis).
		WithSpec(ctx.Spec).
		WithDiff(diff).
		WithExtra(reviewExtra.String()).
		WithLessons(lessonsStr).
		Build(), nil
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
