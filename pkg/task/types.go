package task

import (
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Task struct {
	ID        string            `json:"id"`
	Name      string            `json:"name,omitempty"`      // Task name in Taskfile
	Cmd       string            `json:"cmd,omitempty"`       // Arbitrary command
	Args      []string          `json:"args,omitempty"`      // Arguments for the command
	Vars      map[string]string `json:"vars,omitempty"`      // Variables for Taskfile
	Status    Status            `json:"status"`
	LogPath   string            `json:"log_path"`
	Error     string            `json:"error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	StartedAt *time.Time        `json:"started_at,omitempty"`
	EndedAt   *time.Time        `json:"ended_at,omitempty"`
}

type SubmitTaskRequest struct {
	Name string            `json:"name"`
	Cmd  string            `json:"cmd"`
	Vars map[string]string `json:"vars"`
}

type SubmitTaskResponse struct {
	ID      string `json:"id"`
	LogPath string `json:"log_path"`
}
