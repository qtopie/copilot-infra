package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/qtopie/copilot-infra/pkg/state"
	"github.com/qtopie/copilot-infra/pkg/task"
	"github.com/qtopie/copilot-infra/pkg/worker"
)

type Server struct {
	store  *state.Store
	worker *worker.Worker
	logDir string
}

func NewServer(store *state.Store, worker *worker.Worker, logDir string) *Server {
	return &Server{
		store:  store,
		worker: worker,
		logDir: logDir,
	}
}

func (s *Server) RegisterRoutes(r *gin.Engine) {
	r.POST("/tasks", s.HandleSubmitTask)
	r.GET("/tasks/:id", s.HandleGetTask)
	r.DELETE("/tasks/:id", s.HandleCancelTask)
	r.GET("/tasks/:id/logs", s.HandleGetLogs)
}

func (s *Server) HandleCancelTask(c *gin.Context) {
	id := c.Param("id")
	if ok := s.worker.Cancel(id); ok {
		c.JSON(http.StatusOK, gin.H{"message": "task cancellation signaled"})
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "task not found or not running"})
}

func (s *Server) HandleSubmitTask(c *gin.Context) {
	var req task.SubmitTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskID := uuid.New().String()
	logPath := filepath.Join(s.logDir, fmt.Sprintf("%s.log", taskID))

	t := &task.Task{
		ID:        taskID,
		Name:      req.Name,
		Cmd:       req.Cmd,
		Vars:      req.Vars,
		Status:    task.StatusPending,
		LogPath:   logPath,
		CreatedAt: time.Now(),
	}

	if err := s.store.SaveTask(c.Request.Context(), t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save task"})
		return
	}

	s.worker.Submit(t)

	c.JSON(http.StatusAccepted, task.SubmitTaskResponse{
		ID:      taskID,
		LogPath: logPath,
	})
}

func (s *Server) HandleGetTask(c *gin.Context) {
	id := c.Param("id")
	t, err := s.store.GetTask(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, t)
}

func (s *Server) HandleGetLogs(c *gin.Context) {
	id := c.Param("id")
	t, err := s.store.GetTask(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	content, err := os.ReadFile(t.LogPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read logs"})
		return
	}

	c.Data(http.StatusOK, "text/plain", content)
}
