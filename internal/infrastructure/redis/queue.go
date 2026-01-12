package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	jobQueueKey = "conversion:jobs"
)

type JobQueue interface {
	Push(ctx context.Context, job *domain.ConversionJob) error
	Pop(ctx context.Context, timeout time.Duration) (*domain.ConversionJob, error)
	Len(ctx context.Context) (int64, error)
}

type jobQueue struct {
	rdb *redis.Client
}

func NewJobQueue(client Client) JobQueue {
	return &jobQueue{
		rdb: client.GetRDB(),
	}
}

func (q *jobQueue) Push(ctx context.Context, job *domain.ConversionJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := q.rdb.LPush(ctx, jobQueueKey, data).Err(); err != nil {
		return fmt.Errorf("failed to push job to queue: %w", err)
	}

	return nil
}

func (q *jobQueue) Pop(ctx context.Context, timeout time.Duration) (*domain.ConversionJob, error) {
	result, err := q.rdb.BRPop(ctx, timeout, jobQueueKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to pop job from queue: %w", err)
	}

	if len(result) < 2 {
		return nil, nil
	}

	var job domain.ConversionJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

func (q *jobQueue) Len(ctx context.Context) (int64, error) {
	length, err := q.rdb.LLen(ctx, jobQueueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}
	return length, nil
}
