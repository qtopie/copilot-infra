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

type Type string

const (
	TypeTaskfile Type = "taskfile"
	TypeInfra    Type = "infra"
	TypeBrowser  Type = "browser"
)

type Task struct {
	ID            string            `json:"id"`
	Type          Type              `json:"type"`                // taskfile or infra
	Name          string            `json:"name,omitempty"`      // Task name in Taskfile or Stack name in Pulumi
	Project       string            `json:"project,omitempty"`   // Pulumi project name
	WorkDir       string            `json:"work_dir,omitempty"`  // Working directory for the command
	Cmd           string            `json:"cmd,omitempty"`       // Arbitrary command
	Args          []string          `json:"args,omitempty"`      // Arguments for the command
	Vars          map[string]string `json:"vars,omitempty"`      // Variables for Taskfile or Pulumi Config
	IsLongRunning bool              `json:"is_long_running"`     // True if managed as a long-running daemon by Pulumi
	AccessURL     string            `json:"access_url,omitempty"` // URL to access the task (e.g. browser debug URL)
	Status        Status            `json:"status"`
	LogPath       string            `json:"log_path"`
	Error         string            `json:"error,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	EndedAt       *time.Time        `json:"ended_at,omitempty"`
	Group         string            `json:"group,omitempty"`
	PID           int               `json:"pid,omitempty"`
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
