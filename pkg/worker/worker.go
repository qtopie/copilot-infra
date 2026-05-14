package worker

import (
	"context"
	"log"
	"time"

	"github.com/qtopie/copilot-infra/pkg/state"
	"github.com/qtopie/copilot-infra/pkg/task"
)

type Worker struct {
	executor *task.Executor
	store    *state.Store
	queue    chan *task.Task
}

func NewWorker(executor *task.Executor, store *state.Store) *Worker {
	return &Worker{
		executor: executor,
		store:    store,
		queue:    make(chan *task.Task, 100),
	}
}

func (w *Worker) Submit(t *task.Task) {
	w.queue <- t
}

func (w *Worker) Start(ctx context.Context) {
	log.Println("Background worker started")
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-w.queue:
			w.processTask(ctx, t)
		}
	}
}

func (w *Worker) processTask(ctx context.Context, t *task.Task) {
	log.Printf("Processing task: %s", t.ID)

	now := time.Now()
	t.Status = task.StatusRunning
	t.StartedAt = &now
	w.store.SaveTask(ctx, t)

	err := w.executor.Execute(ctx, t)

	ended := time.Now()
	t.EndedAt = &ended
	if err != nil {
		log.Printf("Task %s failed: %v", t.ID, err)
		t.Status = task.StatusFailed
		t.Error = err.Error()
	} else {
		log.Printf("Task %s succeeded", t.ID)
		t.Status = task.StatusSucceeded
	}

	w.store.SaveTask(ctx, t)
}
