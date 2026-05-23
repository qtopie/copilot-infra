package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dapr/go-sdk/client"
	"github.com/qtopie/copilot-infra/pkg/task"
)

const (
	StoreName     = "statestore"
	TaskIndexKey  = "task_index"
)

type Store struct {
	client client.Client
}

func NewStore(c client.Client) *Store {
	return &Store{client: c}
}

func (s *Store) SaveTask(ctx context.Context, t *task.Task) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	
	// 1. Save task data
	if err := s.client.SaveState(ctx, StoreName, t.ID, data, nil); err != nil {
		return err
	}

	// 2. Update index
	return s.addToIndex(ctx, t.ID)
}

func (s *Store) addToIndex(ctx context.Context, id string) error {
	ids, err := s.getTaskIDs(ctx)
	if err != nil {
		ids = []string{}
	}

	// Check if already in index
	for _, existingID := range ids {
		if existingID == id {
			return nil
		}
	}

	ids = append(ids, id)
	data, _ := json.Marshal(ids)
	return s.client.SaveState(ctx, StoreName, TaskIndexKey, data, nil)
}

func (s *Store) getTaskIDs(ctx context.Context) ([]string, error) {
	item, err := s.client.GetState(ctx, StoreName, TaskIndexKey, nil)
	if err != nil || item == nil || len(item.Value) == 0 {
		return nil, fmt.Errorf("index not found")
	}

	var ids []string
	if err := json.Unmarshal(item.Value, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *Store) ListTasks(ctx context.Context) ([]*task.Task, error) {
	ids, err := s.getTaskIDs(ctx)
	if err != nil {
		return []*task.Task{}, nil // Empty list if no index
	}

	var tasks []*task.Task
	for _, id := range ids {
		t, err := s.GetTask(ctx, id)
		if err == nil {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (s *Store) GetTask(ctx context.Context, id string) (*task.Task, error) {
	item, err := s.client.GetState(ctx, StoreName, id, nil)
	if err != nil {
		return nil, err
	}
	if item == nil || len(item.Value) == 0 {
		return nil, fmt.Errorf("task %s not found", id)
	}

	var t task.Task
	if err := json.Unmarshal(item.Value, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
