package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
	"github.com/openmeterio/openmeter/pkg/models"
)

// keyPrefix is the prefix for progress data in the Redis store.
// All progress keys will be stored as "progress:<namespace>:<id>"
const staticKeyPrefix = "progress:"

// GetProgress retrieves the progress
func (a *adapter) GetProgress(ctx context.Context, input entity.GetProgressInput) (*entity.Progress, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate get progress input: %w", err)
	}

	var progress entity.Progress

	cmd := a.redis.Get(ctx, a.getKey(input.ProgressID))

	if cmd.Err() != nil {
		if cmd.Err() == redis.Nil {
			return nil, models.NewGenericNotFoundError(
				fmt.Errorf("progress not found for id: %s", input.ProgressID.ID),
			)
		}

		return nil, fmt.Errorf("get progress: %w", cmd.Err())
	}

	if err := json.Unmarshal([]byte(cmd.Val()), &progress); err != nil {
		return nil, fmt.Errorf("unmarshal progress: %w", err)
	}

	return &progress, nil
}

// UpsertProgress updates the progress
func (a *adapter) UpsertProgress(ctx context.Context, input entity.UpsertProgressInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate upsert progress input: %w", err)
	}

	data, err := json.Marshal(input.Progress)
	if err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}

	cmd := a.redis.Set(ctx, a.getKey(input.ProgressID), data, a.expiration)
	if cmd.Err() != nil {
		return fmt.Errorf("set progress: %w", cmd.Err())
	}

	return nil
}

// getKey returns the key for the KV store
func (a *adapter) getKey(id entity.ProgressID) string {
	if a.keyPrefix == "" {
		return fmt.Sprintf("%s:%s:%s", staticKeyPrefix, id.Namespace, id.ID)
	}

	return fmt.Sprintf("%s:%s:%s:%s", a.keyPrefix, staticKeyPrefix, id.Namespace, id.ID)
}
