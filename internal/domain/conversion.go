package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type ConversionStatus string

const (
	ConversionStatusPending   ConversionStatus = "PENDING"
	ConversionStatusFetching  ConversionStatus = "FETCHING"
	ConversionStatusMatching  ConversionStatus = "MATCHING"
	ConversionStatusCreating  ConversionStatus = "CREATING"
	ConversionStatusCompleted ConversionStatus = "COMPLETED"
	ConversionStatusFailed    ConversionStatus = "FAILED"
)

func (s ConversionStatus) IsValid() bool {
	switch s {
	case ConversionStatusPending, ConversionStatusFetching, ConversionStatusMatching,
		ConversionStatusCreating, ConversionStatusCompleted, ConversionStatusFailed:
		return true
	default:
		return false
	}
}

func (s ConversionStatus) IsTerminal() bool {
	return s == ConversionStatusCompleted || s == ConversionStatusFailed
}

type Conversion struct {
	ID                 string           `json:"id"`
	UserID             string           `json:"userId"`
	SourcePlatform     Platform         `json:"sourcePlatform"`
	TargetPlatform     Platform         `json:"targetPlatform"`
	SourcePlaylistID   string           `json:"sourcePlaylistId"`
	SourcePlaylistName string           `json:"sourcePlaylistName,omitempty"`
	TargetPlaylistID   string           `json:"targetPlaylistId,omitempty"`
	TargetPlaylistURL  string           `json:"targetPlaylistUrl,omitempty"`
	TargetPlaylistName string           `json:"targetPlaylistName"`
	Status             ConversionStatus `json:"status"`
	TotalTracks        int              `json:"totalTracks"`
	ProcessedTracks    int              `json:"processedTracks"`
	MatchedTracks      int              `json:"matchedTracks"`
	FailedTracks       int              `json:"failedTracks"`
	ErrorMessage       string           `json:"errorMessage,omitempty"`
	CreatedAt          time.Time        `json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
	CompletedAt        *time.Time       `json:"completedAt,omitempty"`
}

type ConversionJob struct {
	JobID              string    `json:"jobId"`
	UserID             string    `json:"userId"`
	SourcePlatform     Platform  `json:"sourcePlatform"`
	TargetPlatform     Platform  `json:"targetPlatform"`
	SourcePlaylistID   string    `json:"sourcePlaylistId"`
	SelectedTrackIDs   []string  `json:"selectedTrackIds,omitempty"`
	TargetPlaylistName string    `json:"targetPlaylistName"`
	CreatedAt          time.Time `json:"createdAt"`
}

func NewConversion(job *ConversionJob) (*Conversion, error) {
	if job == nil {
		return nil, errors.New("job cannot be nil")
	}
	if job.JobID == "" {
		return nil, errors.New("job ID cannot be empty")
	}
	if job.UserID == "" {
		return nil, errors.New("user ID cannot be empty")
	}
	if !job.SourcePlatform.IsValid() {
		return nil, errors.New("invalid source platform")
	}
	if !job.TargetPlatform.IsValid() {
		return nil, errors.New("invalid target platform")
	}
	if job.SourcePlaylistID == "" {
		return nil, errors.New("source playlist ID cannot be empty")
	}

	now := time.Now()
	return &Conversion{
		ID:                 job.JobID,
		UserID:             job.UserID,
		SourcePlatform:     job.SourcePlatform,
		TargetPlatform:     job.TargetPlatform,
		SourcePlaylistID:   job.SourcePlaylistID,
		TargetPlaylistName: job.TargetPlaylistName,
		Status:             ConversionStatusPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func NewConversionJob(userID string, sourcePlatform, targetPlatform Platform, sourcePlaylistID, targetPlaylistName string) *ConversionJob {
	return &ConversionJob{
		JobID:              uuid.New().String(),
		UserID:             userID,
		SourcePlatform:     sourcePlatform,
		TargetPlatform:     targetPlatform,
		SourcePlaylistID:   sourcePlaylistID,
		TargetPlaylistName: targetPlaylistName,
		CreatedAt:          time.Now(),
	}
}

func (c *Conversion) StartFetching() {
	c.Status = ConversionStatusFetching
	c.UpdatedAt = time.Now()
}

func (c *Conversion) StartMatching(totalTracks int, sourcePlaylistName string) {
	c.Status = ConversionStatusMatching
	c.TotalTracks = totalTracks
	c.SourcePlaylistName = sourcePlaylistName
	c.UpdatedAt = time.Now()
}

func (c *Conversion) UpdateProgress(processed, matched, failed int) {
	c.ProcessedTracks = processed
	c.MatchedTracks = matched
	c.FailedTracks = failed
	c.UpdatedAt = time.Now()
}

func (c *Conversion) StartCreating() {
	c.Status = ConversionStatusCreating
	c.UpdatedAt = time.Now()
}

func (c *Conversion) Complete(targetPlaylistID, targetPlaylistURL string) {
	now := time.Now()
	c.Status = ConversionStatusCompleted
	c.TargetPlaylistID = targetPlaylistID
	c.TargetPlaylistURL = targetPlaylistURL
	c.UpdatedAt = now
	c.CompletedAt = &now
}

func (c *Conversion) Fail(errorMessage string) {
	now := time.Now()
	c.Status = ConversionStatusFailed
	c.ErrorMessage = errorMessage
	c.UpdatedAt = now
	c.CompletedAt = &now
}

func (c *Conversion) Progress() int {
	if c.TotalTracks == 0 {
		return 0
	}
	return (c.ProcessedTracks * 100) / c.TotalTracks
}

func (c *Conversion) EstimatedSecondsRemaining(avgSecondsPerTrack float64) int {
	remaining := c.TotalTracks - c.ProcessedTracks
	if remaining <= 0 {
		return 0
	}
	return int(float64(remaining) * avgSecondsPerTrack)
}
