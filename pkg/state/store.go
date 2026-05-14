package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dapr/go-sdk/client"
	"github.com/qtopie/copilot-infra/pkg/task"
)

const StoreName = "statestore"

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
	return s.client.SaveState(ctx, StoreName, t.ID, data, nil)
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
