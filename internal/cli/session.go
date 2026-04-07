package cli

import (
	"github.com/nqode/qode/internal/config"
	gocontext "github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/git"
	"github.com/nqode/qode/internal/prompt"
)

// Session holds the common bootstrapped state used by most CLI commands.
type Session struct {
	Root    string
	Config  *config.Config
	Branch  string
	Context *gocontext.Context
	Engine  *prompt.Engine
}

func loadSession() (*Session, error) {
	root, err := resolveRoot()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return nil, err
	}
	branch, err := git.CurrentBranch(root)
	if err != nil {
		return nil, err
	}
	ctx, err := gocontext.Load(root, branch)
	if err != nil {
		return nil, err
	}
	engine, err := prompt.NewEngine(root)
	if err != nil {
		return nil, err
	}
	return &Session{Root: root, Config: cfg, Branch: branch, Context: ctx, Engine: engine}, nil
}
