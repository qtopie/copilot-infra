package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	gotask "github.com/go-task/task/v3"
	"github.com/go-task/task/v3/taskfile/ast"
)

type Executor struct {
	LogDir string
}

func NewExecutor(logDir string) (*Executor, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	return &Executor{LogDir: logDir}, nil
}

func (e *Executor) Execute(ctx context.Context, t *Task) error {
	logFile, err := os.Create(t.LogPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(logFile, os.Stdout)

	if t.Cmd != "" {
		return e.executeRawCommand(ctx, t, multiWriter)
	}
	return e.executeTaskfile(ctx, t, multiWriter)
}

func (e *Executor) executeRawCommand(ctx context.Context, t *Task, w io.Writer) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", t.Cmd)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func (e *Executor) executeTaskfile(ctx context.Context, t *Task, w io.Writer) error {
	executor := gotask.NewExecutor(
		gotask.WithStdout(w),
		gotask.WithStderr(w),
	)

	if err := executor.Setup(); err != nil {
		return fmt.Errorf("failed to setup task executor: %w", err)
	}

	vars := &ast.Vars{}
	for k, v := range t.Vars {
		vars.Set(k, ast.Var{Value: v})
	}

	return executor.Run(ctx, &gotask.Call{
		Task: t.Name,
		Vars: vars,
	})
}
