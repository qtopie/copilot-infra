package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"syscall"

	gotask "github.com/go-task/task/v3"
	"github.com/go-task/task/v3/taskfile/ast"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/qtopie/copilot-infra/pkg/infra"
)

type Store interface {
	SaveTask(ctx context.Context, t *Task) error
}

type Executor struct {
	LogDir       string
	InfraManager *infra.Manager
	Store        Store
}

func NewExecutor(logDir string, infraManager *infra.Manager, store Store) (*Executor, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	return &Executor{
		LogDir:       logDir,
		InfraManager: infraManager,
		Store:        store,
	}, nil
}

func (e *Executor) Execute(ctx context.Context, t *Task) error {
	logFile, err := os.OpenFile(t.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(logFile, os.Stdout)

	if t.IsLongRunning || t.Type == TypeBrowser {
		return e.executeLongTaskViaPulumi(ctx, t, logFile)
	}

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

func (e *Executor) executeLongTaskViaPulumi(ctx context.Context, t *Task, logFile *os.File) error {
	if e.InfraManager == nil {
		return fmt.Errorf("infra manager not initialized")
	}

	proj := t.Project
	if proj == "" {
		proj = "copilot-infra-daemons"
	}
	stackName := t.ID

	var program pulumi.RunFunc

	if t.Type == TypeBrowser {
		port := 9222
		if p, ok := t.Vars["PORT"]; ok {
			fmt.Sscanf(p, "%d", &port)
		}
		userDataDir := filepath.Join(os.TempDir(), "copilot-infra-browser", t.ID)
		if d, ok := t.Vars["USER_DATA_DIR"]; ok {
			userDataDir = d
		}
		program = infra.BrowserProgram(port, userDataDir, t.LogPath)
	} else {
		program = func(pCtx *pulumi.Context) error {
			// Create a background command that redirects output to the log file and saves its PID
			pidFile := t.LogPath + ".pid"
			
			command := t.Cmd
			if t.WorkDir != "" {
				command = fmt.Sprintf("cd %s && %s", t.WorkDir, t.Cmd)
			}
			
			// Using bash/sh to daemonize
			createCmd := fmt.Sprintf("nohup sh -c '%s' >> %s 2>&1 & echo $! > %s", command, t.LogPath, pidFile)
			fmt.Printf("[Executor] Pulumi createCmd: %s\n", createCmd)
			deleteCmd := fmt.Sprintf("kill -9 $(cat %s) || true; rm -f %s", pidFile, pidFile)

			_, err := local.NewCommand(pCtx, "daemon-"+t.ID, &local.CommandArgs{
				Create: pulumi.String(createCmd),
				Delete: pulumi.String(deleteCmd),
			})

			if err == nil {
				pCtx.Export("task_id", pulumi.String(t.ID))
				pCtx.Export("command", pulumi.String(t.Cmd))
				pCtx.Export("pid_file", pulumi.String(pidFile))
			}
			return err
		}
	}

	err := e.InfraManager.Deploy(ctx, proj, stackName, program, logFile)
	if err != nil {
		return err
	}

	// Try to get outputs
	outputs, err := e.InfraManager.GetOutputs(ctx, proj, stackName)
	if err == nil {
		if val, ok := outputs["debug_url"]; ok {
			if s, ok := val.Value.(string); ok {
				t.AccessURL = s
			}
		}
	}

	return nil
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
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if t.WorkDir != "" {
		cmd.Dir = t.WorkDir
	}
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		return err
	}

	t.PID = cmd.Process.Pid
	if e.Store != nil {
		_ = e.Store.SaveTask(ctx, t)
	}

	return cmd.Wait()
}

func (e *Executor) executeTaskfile(ctx context.Context, t *Task, w io.Writer) error {
	opts := []gotask.ExecutorOption{
		gotask.WithStdout(w),
		gotask.WithStderr(w),
	}
	if t.WorkDir != "" {
		opts = append(opts, gotask.WithDir(t.WorkDir))
	}

	executor := gotask.NewExecutor(opts...)

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
