package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	gotask "github.com/go-task/task/v3"
	"github.com/go-task/task/v3/taskfile/ast"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/qtopie/copilot-infra/pkg/infra"
)

type Executor struct {
	LogDir       string
	InfraManager *infra.Manager
}

func NewExecutor(logDir string, infraManager *infra.Manager) (*Executor, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	return &Executor{
		LogDir:       logDir,
		InfraManager: infraManager,
	}, nil
}

func (e *Executor) Execute(ctx context.Context, t *Task) error {
	logFile, err := os.OpenFile(t.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(logFile, os.Stdout)

	// --- Persistence Logic ---
	// If the task type is 'infra', or the command starts with 'pulumi', 
	// we use the Pulumi Automation API to ensure persistence.
	if t.Type == TypeInfra || strings.HasPrefix(strings.TrimSpace(t.Cmd), "pulumi ") {
		return e.executeInfra(ctx, t, logFile)
	}

	if t.Cmd != "" {
		return e.executeRawCommand(ctx, t, multiWriter)
	}
	return e.executeTaskfile(ctx, t, multiWriter)
}

func (e *Executor) executeInfra(ctx context.Context, t *Task, logFile *os.File) error {
	if e.InfraManager == nil {
		return fmt.Errorf("infra manager not initialized")
	}

	// For the prototype, we use a simple program that can be configured via vars
	// In a real scenario, this would be more dynamic.
	program := func(pCtx *pulumi.Context) error {
		// Example: Just an output for now
		pCtx.Export("task_id", pulumi.String(t.ID))
		pCtx.Export("command", pulumi.String(t.Cmd))
		return nil
	}

	return e.InfraManager.Deploy(ctx, t.Project, t.Name, program, logFile)
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
