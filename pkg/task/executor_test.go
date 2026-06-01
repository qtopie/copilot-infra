package task

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutor_ExecuteRawCommand(t *testing.T) {
	logDir := ".test_logs"
	executor, err := NewExecutor(logDir, nil)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	defer os.RemoveAll(logDir)

	taskID := "test-raw"
	logPath := filepath.Join(logDir, taskID+".log")
	tk := &Task{
		ID:      taskID,
		Cmd:     "echo 'hello world'",
		LogPath: logPath,
	}

	err = executor.Execute(context.Background(), tk)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	if !strings.Contains(string(content), "hello world") {
		t.Errorf("log does not contain expected output: %s", string(content))
	}
}

func TestExecutor_ExecuteTaskfile(t *testing.T) {
	// Create a temporary Taskfile.yml
	taskfileContent := `version: '3'
tasks:
  hello:
    cmds:
      - echo "hello from taskfile"
`
	err := os.WriteFile("Taskfile.yml", []byte(taskfileContent), 0644)
	if err != nil {
		t.Fatalf("failed to create Taskfile: %v", err)
	}
	defer os.Remove("Taskfile.yml")

	logDir := ".test_logs_task"
	executor, err := NewExecutor(logDir, nil)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	defer os.RemoveAll(logDir)

	taskID := "test-taskfile"
	logPath := filepath.Join(logDir, taskID+".log")
	tk := &Task{
		ID:      taskID,
		Name:    "hello",
		LogPath: logPath,
	}

	err = executor.Execute(context.Background(), tk)
	if err != nil {
		t.Errorf("execution failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	if !strings.Contains(string(content), "hello from taskfile") {
		t.Errorf("log does not contain expected output: %s", string(content))
	}
}
