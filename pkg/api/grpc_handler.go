package api

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	taskv1 "github.com/qtopie/copilot-infra/pkg/api/proto/v1"
	"github.com/qtopie/copilot-infra/pkg/state"
	"github.com/qtopie/copilot-infra/pkg/task"
	"github.com/qtopie/copilot-infra/pkg/worker"
	"github.com/qtopie/sniphunt/pkg/search"
)

type GRPCHandler struct {
	taskv1.UnimplementedTaskServiceServer
	worker        *worker.Worker
	store         *state.Store
	logDir        string
	componentsDir string
}

func NewGRPCHandler(worker *worker.Worker, store *state.Store, logDir, componentsDir string) *GRPCHandler {
	return &GRPCHandler{
		worker:        worker,
		store:         store,
		logDir:        logDir,
		componentsDir: componentsDir,
	}
}

func (h *GRPCHandler) SubmitTask(ctx context.Context, req *taskv1.SubmitTaskRequest) (*taskv1.SubmitTaskResponse, error) {
	var id string
	var existing *task.Task

	// 1. Determine ID: Use deterministic UUID if name is provided, else random
	if req.Name != "" {
		id = uuid.NewMD5(uuid.NameSpaceDNS, []byte(req.Name)).String()
		fmt.Printf("[GRPCHandler] Using deterministic ID %s for task name: %s\n", id, req.Name)
		
		// Check if it already exists
		existing, _ = h.store.GetTask(ctx, id)
	} else {
		id = uuid.New().String()
	}

	// 2. If exists, cancel and prepare for restart
	if existing != nil {
		fmt.Printf("[GRPCHandler] Task %s already exists, restarting...\n", req.Name)
		h.worker.Cancel(id)
		
		// Update dynamic fields
		if req.Command != "" {
			existing.Cmd = req.Command
		}
		if req.Env != nil {
			existing.Vars = req.Env
		}
		if req.WorkDir != "" {
			existing.WorkDir = req.WorkDir
		}
		existing.Status = task.StatusPending
		existing.Error = ""
		existing.StartedAt = nil
		existing.EndedAt = nil
		
		if err := h.store.SaveTask(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update task: %w", err)
		}
		
		h.worker.Submit(existing)
		return &taskv1.SubmitTaskResponse{TaskId: id}, nil
	}

	// 3. Create new task
	taskType := task.TypeTaskfile
	switch req.Type {
	case "infra":
		taskType = task.TypeInfra
	case "browser":
		taskType = task.TypeBrowser
	}

	t := &task.Task{
		ID:        id,
		Type:      taskType,
		Name:      req.Name,
		Project:   req.Project,
		WorkDir:   req.WorkDir,
		Cmd:       req.Command,
		Vars:      req.Env,
		Status:    task.StatusPending,
		LogPath:   fmt.Sprintf("%s/%s.log", h.logDir, id),
		CreatedAt: time.Now(),
	}

	if err := h.store.SaveTask(ctx, t); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	h.worker.Submit(t)

	return &taskv1.SubmitTaskResponse{
		TaskId: id,
	}, nil
}

func (h *GRPCHandler) mapTaskToResponse(t *task.Task) *taskv1.GetTaskResponse {
	return &taskv1.GetTaskResponse{
		TaskId:    t.ID,
		Status:    string(t.Status),
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.CreatedAt.Format(time.RFC3339), // Simplified
		Error:     t.Error,
		AccessUrl: t.AccessURL,
		Name:      t.Name,
		Command:   t.Cmd,
		Type:      string(t.Type),
		WorkDir:   t.WorkDir,
	}
}

func (h *GRPCHandler) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	t, err := h.store.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	return h.mapTaskToResponse(t), nil
}

func (h *GRPCHandler) GetTaskLogs(ctx context.Context, req *taskv1.GetTaskLogsRequest) (*taskv1.GetTaskLogsResponse, error) {
	t, err := h.store.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(t.LogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	return &taskv1.GetTaskLogsResponse{
		Content: string(content),
	}, nil
}

func (h *GRPCHandler) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (*taskv1.ListTasksResponse, error) {
	tasks, err := h.store.ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	var respTasks []*taskv1.GetTaskResponse
	for _, t := range tasks {
		respTasks = append(respTasks, h.mapTaskToResponse(t))
	}

	return &taskv1.ListTasksResponse{Tasks: respTasks}, nil
}

func (h *GRPCHandler) RestartTask(ctx context.Context, req *taskv1.RestartTaskRequest) (*taskv1.RestartTaskResponse, error) {
	t, err := h.store.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	h.worker.Cancel(req.TaskId)

	t.Status = task.StatusPending
	t.Error = ""
	t.StartedAt = nil
	t.EndedAt = nil

	if err := h.store.SaveTask(ctx, t); err != nil {
		return nil, err
	}

	h.worker.Submit(t)
	return &taskv1.RestartTaskResponse{TaskId: t.ID}, nil
}

func (h *GRPCHandler) CancelTask(ctx context.Context, req *taskv1.CancelTaskRequest) (*taskv1.CancelTaskResponse, error) {
	t, err := h.store.GetTask(ctx, req.TaskId)
	if err == nil && t.IsLongRunning {
		// Destroy the background task via Pulumi
		proj := t.Project
		if proj == "" {
			proj = "copilot-infra-daemons"
		}
		if h.worker.Executor().InfraManager != nil {
			err = h.worker.Executor().InfraManager.Destroy(ctx, proj, t.ID, os.Stdout)
			if err != nil {
				return nil, fmt.Errorf("failed to destroy long-running task: %w", err)
			}
			t.Status = task.StatusCancelled
			h.store.SaveTask(ctx, t)
			return &taskv1.CancelTaskResponse{Message: "long-running task destroyed"}, nil
		}
	}

	if ok := h.worker.Cancel(req.TaskId); ok {
		return &taskv1.CancelTaskResponse{Message: "task cancellation signaled"}, nil
	}
	return nil, fmt.Errorf("task not found or not running")
}

func (h *GRPCHandler) RegisterConnection(ctx context.Context, req *taskv1.RegisterConnectionRequest) (*taskv1.RegisterConnectionResponse, error) {
	metadata := fmt.Sprintf(`  - name: url
    value: "%s"`, req.Url)

	if req.Ns != "" {
		metadata += fmt.Sprintf("\n  - name: \"header.surreal-ns\"\n    value: \"%s\"", req.Ns)
	}
	if req.Db != "" {
		metadata += fmt.Sprintf("\n  - name: \"header.surreal-db\"\n    value: \"%s\"", req.Db)
	}
	if req.Auth != "" {
		metadata += fmt.Sprintf("\n  - name: \"header.Authorization\"\n    value: \"%s\"", req.Auth)
	}

	content := fmt.Sprintf(`apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: %s
spec:
  type: bindings.http
  version: v1
  metadata:
%s
`, req.Name, metadata)

	filePath := filepath.Join(h.componentsDir, fmt.Sprintf("%s.yaml", req.Name))
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to save connection component: %w", err)
	}

	return &taskv1.RegisterConnectionResponse{
		Message:   "connection registered",
		Name:      req.Name,
		DaprPath: fmt.Sprintf("/v1.0/bindings/%s", req.Name),
	}, nil
}

func (h *GRPCHandler) SearchLogs(ctx context.Context, req *taskv1.SearchLogsRequest) (*taskv1.SearchResponse, error) {
	t, err := h.store.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	searcher := search.NewSearcher()
	dir := filepath.Dir(t.LogPath)
	filename := filepath.Base(t.LogPath)

	matchChan, errChan := searcher.Search(ctx, dir, req.Pattern)

	var matches []*taskv1.SearchMatch
	for {
		select {
		case match, ok := <-matchChan:
			if !ok {
				matchChan = nil
			} else {
				if filepath.Base(match.Path) == filename {
					m := &taskv1.SearchMatch{
						Path:    match.Path,
						LineNum: int32(match.LineNum),
						Text:    string(match.Text),
					}
					if req.Before > 0 || req.After > 0 {
						h.fillContext(m, int(req.Before), int(req.After))
					}
					matches = append(matches, m)
				}
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
			} else if err != nil {
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if matchChan == nil && errChan == nil {
			break
		}
	}

	return &taskv1.SearchResponse{Matches: matches}, nil
}

func (h *GRPCHandler) GlobalSearch(ctx context.Context, req *taskv1.GlobalSearchRequest) (*taskv1.SearchResponse, error) {
	searcher := search.NewSearcher()
	if req.Ext != "" {
		searcher.Extensions = []string{req.Ext}
	}

	pathInfo, err := os.Stat(req.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	searchDir := req.Path
	var filterFile string
	if !pathInfo.IsDir() {
		searchDir = filepath.Dir(req.Path)
		filterFile = filepath.Base(req.Path)
		searcher.Extensions = nil
	}

	matchChan, errChan := searcher.Search(ctx, searchDir, req.Pattern)

	var matches []*taskv1.SearchMatch
	for {
		select {
		case match, ok := <-matchChan:
			if !ok {
				matchChan = nil
			} else {
				if filterFile == "" || filepath.Base(match.Path) == filterFile {
					m := &taskv1.SearchMatch{
						Path:    match.Path,
						LineNum: int32(match.LineNum),
						Text:    string(match.Text),
					}
					if req.Before > 0 || req.After > 0 {
						h.fillContext(m, int(req.Before), int(req.After))
					}
					matches = append(matches, m)
				}
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
			} else if err != nil {
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if matchChan == nil && errChan == nil {
			break
		}
	}

	return &taskv1.SearchResponse{Matches: matches}, nil
}

func (h *GRPCHandler) fillContext(m *taskv1.SearchMatch, before, after int) {
	file, err := os.Open(m.Path)
	if err != nil {
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	currentLine := 0
	start := int(m.LineNum) - before
	if start < 1 {
		start = 1
	}
	end := int(m.LineNum) + after

	for scanner.Scan() {
		currentLine++
		if currentLine >= start && currentLine <= end {
			lines = append(lines, scanner.Text())
		}
		if currentLine > end {
			break
		}
	}

	matchIdx := int(m.LineNum) - start
	if matchIdx >= 0 && matchIdx < len(lines) {
		if before > 0 {
			m.ContextBefore = lines[:matchIdx]
		}
		if after > 0 && matchIdx+1 < len(lines) {
			m.ContextAfter = lines[matchIdx+1:]
		}
	}
}
