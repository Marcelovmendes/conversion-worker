package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConversionStep string

const (
	StepFetchSourcePlaylist  ConversionStep = "FETCH_SOURCE_PLAYLIST"
	StepMatchTrack           ConversionStep = "MATCH_TRACK"
	StepCreateTargetPlaylist ConversionStep = "CREATE_TARGET_PLAYLIST"
	StepAddTrackToPlaylist   ConversionStep = "ADD_TRACK_TO_PLAYLIST"
)

type LogStatus string

const (
	LogStatusSuccess LogStatus = "SUCCESS"
	LogStatusFailed  LogStatus = "FAILED"
	LogStatusSkipped LogStatus = "SKIPPED"
)

type ConversionLog struct {
	ID                string         `json:"id"`
	ConversionID      string         `json:"conversionId"`
	Step              ConversionStep `json:"step"`
	Status            LogStatus      `json:"status"`
	SourceTrackID     string         `json:"sourceTrackId,omitempty"`
	SourceTrackName   string         `json:"sourceTrackName,omitempty"`
	SourceTrackArtist string         `json:"sourceTrackArtist,omitempty"`
	TargetTrackID     string         `json:"targetTrackId,omitempty"`
	TargetTrackName   string         `json:"targetTrackName,omitempty"`
	ErrorMessage      string         `json:"errorMessage,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
}

func newConversionLog(conversionID string, step ConversionStep, status LogStatus) *ConversionLog {
	return &ConversionLog{
		ID:           uuid.New().String(),
		ConversionID: conversionID,
		Step:         step,
		Status:       status,
		CreatedAt:    time.Now(),
	}
}

func NewFetchPlaylistLog(conversionID string, status LogStatus, errorMessage string) *ConversionLog {
	log := newConversionLog(conversionID, StepFetchSourcePlaylist, status)
	log.ErrorMessage = errorMessage
	return log
}

func NewMatchTrackLog(conversionID string, sourceTrack *Track, targetTrack *Track, status LogStatus) *ConversionLog {
	log := newConversionLog(conversionID, StepMatchTrack, status)

	if sourceTrack != nil {
		log.SourceTrackID = sourceTrack.PlatformID
		log.SourceTrackName = sourceTrack.Name
		log.SourceTrackArtist = sourceTrack.Artist
	}

	if targetTrack != nil {
		log.TargetTrackID = targetTrack.PlatformID
		log.TargetTrackName = targetTrack.Name
	}

	return log
}

func NewMatchTrackErrorLog(conversionID string, sourceTrack *Track, errorMessage string) *ConversionLog {
	log := NewMatchTrackLog(conversionID, sourceTrack, nil, LogStatusFailed)
	log.ErrorMessage = errorMessage
	return log
}

func NewCreatePlaylistLog(conversionID string, status LogStatus, errorMessage string) *ConversionLog {
	log := newConversionLog(conversionID, StepCreateTargetPlaylist, status)
	log.ErrorMessage = errorMessage
	return log
}

func NewAddTrackLog(conversionID string, targetTrack *Track, status LogStatus, errorMessage string) *ConversionLog {
	log := newConversionLog(conversionID, StepAddTrackToPlaylist, status)

	if targetTrack != nil {
		log.TargetTrackID = targetTrack.PlatformID
		log.TargetTrackName = targetTrack.Name
	}

	log.ErrorMessage = errorMessage
	return log
}
