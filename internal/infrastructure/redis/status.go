package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	statusKeyPrefix = "conversion:"
	statusKeySuffix = ":status"
	statusTTL       = 24 * time.Hour
)

type ConversionStatusData struct {
	JobID                     string                   `json:"jobId"`
	Status                    domain.ConversionStatus  `json:"status"`
	Progress                  int                      `json:"progress"`
	TotalTracks               int                      `json:"totalTracks"`
	ProcessedTracks           int                      `json:"processedTracks"`
	MatchedTracks             int                      `json:"matchedTracks"`
	FailedTracks              int                      `json:"failedTracks"`
	EstimatedSecondsRemaining int                      `json:"estimatedSecondsRemaining"`
	TargetPlaylistURL         string                   `json:"targetPlaylistUrl,omitempty"`
	Error                     string                   `json:"error,omitempty"`
	UpdatedAt                 time.Time                `json:"updatedAt"`
}

type StatusStore interface {
	Set(ctx context.Context, status *ConversionStatusData) error
	Get(ctx context.Context, jobID string) (*ConversionStatusData, error)
	Delete(ctx context.Context, jobID string) error
}

type statusStore struct {
	rdb *redis.Client
}

func NewStatusStore(client Client) StatusStore {
	return &statusStore{rdb: client.GetRDB()}
}

func statusKey(jobID string) string {
	return statusKeyPrefix + jobID + statusKeySuffix
}

func (s *statusStore) Set(ctx context.Context, status *ConversionStatusData) error {
	status.UpdatedAt = time.Now()

	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	key := statusKey(status.JobID)
	if err := s.rdb.Set(ctx, key, data, statusTTL).Err(); err != nil {
		return fmt.Errorf("failed to set status: %w", err)
	}

	return nil
}

func (s *statusStore) Get(ctx context.Context, jobID string) (*ConversionStatusData, error) {
	key := statusKey(jobID)

	data, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	var status ConversionStatusData
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

func (s *statusStore) Delete(ctx context.Context, jobID string) error {
	key := statusKey(jobID)
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete status: %w", err)
	}
	return nil
}

func NewStatusFromConversion(c *domain.Conversion) *ConversionStatusData {
	status := &ConversionStatusData{
		JobID:           c.ID,
		Status:          c.Status,
		Progress:        c.Progress(),
		TotalTracks:     c.TotalTracks,
		ProcessedTracks: c.ProcessedTracks,
		MatchedTracks:   c.MatchedTracks,
		FailedTracks:    c.FailedTracks,
		UpdatedAt:       c.UpdatedAt,
	}

	if c.TargetPlaylistURL != "" {
		status.TargetPlaylistURL = c.TargetPlaylistURL
	}

	if c.ErrorMessage != "" {
		status.Error = c.ErrorMessage
	}

	return status
}
