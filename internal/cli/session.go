package cli

import (
	"context"
	"io"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/qodecontext"
)

// Session holds the common bootstrapped state used by most CLI commands.
type Session struct {
	Root    string
	Config  *config.Config
	Context *qodecontext.Context
	Engine  prompt.Renderer
}

func loadSession(errOut io.Writer) (*Session, error) {
	return loadSessionCtx(context.Background(), errOut)
}

func loadSessionCtx(ctx context.Context, errOut io.Writer) (*Session, error) {
	root, err := resolveRoot()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg, err := loadConfigNotifying(errOut, root)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	qctx, err := qodecontext.Load(ctx, root)
	if err != nil {
		return nil, err
	}
	engine, err := prompt.NewEngine(root)
	if err != nil {
		return nil, err
	}
	return &Session{Root: root, Config: cfg, Context: qctx, Engine: engine}, nil
}
