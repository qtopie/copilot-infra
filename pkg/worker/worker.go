package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/qtopie/copilot-infra/pkg/state"
	"github.com/qtopie/copilot-infra/pkg/task"
)

type Worker struct {
	executor *task.Executor
	store    *state.Store
	queue    chan *task.Task
	running  map[string]context.CancelFunc
	mu       sync.Mutex
}

func NewWorker(executor *task.Executor, store *state.Store) *Worker {
	return &Worker{
		executor: executor,
		store:    store,
		queue:    make(chan *task.Task, 100),
		running:  make(map[string]context.CancelFunc),
	}
}

func (w *Worker) Executor() *task.Executor {
	return w.executor
}

func (w *Worker) Submit(t *task.Task) {
	w.queue <- t
}

func (w *Worker) Cancel(id string) bool {
	w.mu.Lock()
	cancel, ok := w.running[id]
	w.mu.Unlock()

	if ok {
		cancel()
		return true
	}
	return false
}

func (w *Worker) Start(ctx context.Context) {
	log.Println("Background worker started")
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-w.queue:
			go w.processTask(ctx, t)
		}
	}
}

func (w *Worker) processTask(ctx context.Context, t *task.Task) {
	log.Printf("Processing task: %s", t.ID)

	taskCtx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.running[t.ID] = cancel
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.running, t.ID)
		w.mu.Unlock()
		cancel()
	}()

	now := time.Now()
	t.Status = task.StatusRunning
	t.StartedAt = &now
	w.store.SaveTask(ctx, t)

	err := w.executor.Execute(taskCtx, t)

	ended := time.Now()
	t.EndedAt = &ended

	if err != nil {
		if taskCtx.Err() == context.Canceled {
			log.Printf("Task %s cancelled", t.ID)
			t.Status = task.StatusCancelled
		} else {
			log.Printf("Task %s failed: %v", t.ID, err)
			t.Status = task.StatusFailed
			t.Error = err.Error()
		}
	} else {
		if t.IsLongRunning {
			log.Printf("Long-running Task %s deployed successfully", t.ID)
			// Do not change status; it remains Running
		} else {
			log.Printf("Task %s succeeded", t.ID)
			t.Status = task.StatusSucceeded
		}
	}

	w.store.SaveTask(ctx, t)
}
