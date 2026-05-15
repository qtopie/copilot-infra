package infra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Manager struct {
	WorkDir string
}

func NewManager(workDir string) (*Manager, error) {
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}
	return &Manager{WorkDir: workDir}, nil
}

// Deploy runs 'pulumi up' for a given project and stack.
// It uses an inline program for this prototype.
func (m *Manager) Deploy(ctx context.Context, projectName, stackName string, deployFunc pulumi.RunFunc, logFile *os.File) error {
	// Setup the workspace and stack
	opts := []auto.LocalWorkspaceOption{
		auto.WorkDir(filepath.Join(m.WorkDir, projectName)),
	}

	s, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, deployFunc, opts...)
	if err != nil {
		return fmt.Errorf("failed to setup stack: %w", err)
	}

	// Run the update
	stdout := os.Stdout
	if logFile != nil {
		stdout = logFile
	}

	_, err = s.Up(ctx, optup.ProgressStreams(stdout))
	if err != nil {
		return fmt.Errorf("failed to run pulumi up: %w", err)
	}

	return nil
}

// Destroy runs 'pulumi destroy'
func (m *Manager) Destroy(ctx context.Context, projectName, stackName string, logFile *os.File) error {
	opts := []auto.LocalWorkspaceOption{
		auto.WorkDir(filepath.Join(m.WorkDir, projectName)),
	}

	s, err := auto.SelectStackInlineSource(ctx, stackName, projectName, func(pCtx *pulumi.Context) error { return nil }, opts...)
	if err != nil {
		return fmt.Errorf("failed to select stack: %w", err)
	}

	stdout := os.Stdout
	if logFile != nil {
		stdout = logFile
	}

	_, err = s.Destroy(ctx, optdestroy.ProgressStreams(stdout))
	return err
}

// GetOutputs retrieves stack outputs
func (m *Manager) GetOutputs(ctx context.Context, projectName, stackName string) (auto.OutputMap, error) {
	opts := []auto.LocalWorkspaceOption{
		auto.WorkDir(filepath.Join(m.WorkDir, projectName)),
	}

	s, err := auto.SelectStackInlineSource(ctx, stackName, projectName, func(pCtx *pulumi.Context) error { return nil }, opts...)
	if err != nil {
		return nil, err
	}

	return s.Outputs(ctx)
}
