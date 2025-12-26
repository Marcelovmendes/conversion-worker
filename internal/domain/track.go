package domain

import (
	"errors"

	"github.com/google/uuid"
)

type Track struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Artist     string   `json:"artist"`
	Album      string   `json:"album"`
	DurationMs int      `json:"durationMs"`
	ISRC       string   `json:"isrc,omitempty"`
	Platform   Platform `json:"platform"`
	PlatformID string   `json:"platformId"`
}

func NewTrack(name, artist string, platform Platform, platformID string) (*Track, error) {
	if name == "" {
		return nil, errors.New("track name cannot be empty")
	}
	if artist == "" {
		return nil, errors.New("track artist cannot be empty")
	}
	if !platform.IsValid() {
		return nil, errors.New("invalid platform")
	}
	if platformID == "" {
		return nil, errors.New("platform ID cannot be empty")
	}

	return &Track{
		ID:         uuid.New().String(),
		Name:       name,
		Artist:     artist,
		Platform:   platform,
		PlatformID: platformID,
	}, nil
}

func (t *Track) WithAlbum(album string) *Track {
	t.Album = album
	return t
}

func (t *Track) WithDuration(durationMs int) *Track {
	t.DurationMs = durationMs
	return t
}

func (t *Track) WithISRC(isrc string) *Track {
	t.ISRC = isrc
	return t
}

type MatchConfidence string

const (
	MatchConfidenceHigh   MatchConfidence = "HIGH"
	MatchConfidenceMedium MatchConfidence = "MEDIUM"
	MatchConfidenceLow    MatchConfidence = "LOW"
	MatchConfidenceNone   MatchConfidence = "NONE"
)

type TrackMatch struct {
	SourceTrack *Track          `json:"sourceTrack"`
	TargetTrack *Track          `json:"targetTrack,omitempty"`
	Confidence  MatchConfidence `json:"confidence"`
	MatchMethod string          `json:"matchMethod,omitempty"`
	Error       string          `json:"error,omitempty"`
}

func NewTrackMatch(source *Track, target *Track, confidence MatchConfidence, method string) *TrackMatch {
	return &TrackMatch{
		SourceTrack: source,
		TargetTrack: target,
		Confidence:  confidence,
		MatchMethod: method,
	}
}

func NewFailedMatch(source *Track, err string) *TrackMatch {
	return &TrackMatch{
		SourceTrack: source,
		Confidence:  MatchConfidenceNone,
		Error:       err,
	}
}
