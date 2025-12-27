package application

import (
	"context"
	"testing"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type mockYouTubeClient struct {
	searchResults map[string]*domain.Track
	searchError   error
}

func (m *mockYouTubeClient) SearchTrack(ctx context.Context, trackName, artistName, sessionID string) (*domain.Track, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	key := trackName + "|" + artistName
	return m.searchResults[key], nil
}

func (m *mockYouTubeClient) CreatePlaylist(ctx context.Context, name, description, sessionID string) (string, string, error) {
	return "playlist-id", "https://youtube.com/playlist?list=xxx", nil
}

func (m *mockYouTubeClient) AddVideosToPlaylist(ctx context.Context, playlistID string, videoIDs []string, sessionID string) error {
	return nil
}

func TestMatcher_MatchTracks(t *testing.T) {
	mockClient := &mockYouTubeClient{
		searchResults: map[string]*domain.Track{},
	}

	track1, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")
	track2, _ := domain.NewTrack("Under Pressure", "Queen", domain.PlatformSpotify, "sp2")

	ytTrack1, _ := domain.NewTrack("Queen - Bohemian Rhapsody (Official Video)", "Queen Official", domain.PlatformYouTube, "yt1")
	ytTrack2, _ := domain.NewTrack("Under Pressure - Queen", "Queen", domain.PlatformYouTube, "yt2")

	mockClient.searchResults["Bohemian Rhapsody|Queen"] = ytTrack1
	mockClient.searchResults["Under Pressure|Queen"] = ytTrack2

	matcher := NewMatcher(mockClient)

	tracks := []*domain.Track{track1, track2}
	var progressCalls int

	matches := matcher.MatchTracks(context.Background(), tracks, "session", 2, func(processed, matched, failed int) {
		progressCalls++
	})

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	if progressCalls != 2 {
		t.Errorf("expected 2 progress calls, got %d", progressCalls)
	}

	for _, match := range matches {
		if match.Confidence == domain.MatchConfidenceNone {
			t.Errorf("expected match for %s, got none", match.SourceTrack.Name)
		}
	}
}

func TestMatcher_NoResults(t *testing.T) {
	mockClient := &mockYouTubeClient{
		searchResults: map[string]*domain.Track{},
	}

	matcher := NewMatcher(mockClient)

	track, _ := domain.NewTrack("Unknown Song", "Unknown Artist", domain.PlatformSpotify, "sp1")
	tracks := []*domain.Track{track}

	matches := matcher.MatchTracks(context.Background(), tracks, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceNone {
		t.Errorf("expected no match, got confidence %v", matches[0].Confidence)
	}

	if matches[0].Error == "" {
		t.Error("expected error message for failed match")
	}
}

func TestMatcher_EmptyTracks(t *testing.T) {
	mockClient := &mockYouTubeClient{}
	matcher := NewMatcher(mockClient)

	matches := matcher.MatchTracks(context.Background(), nil, "session", 1, nil)

	if matches != nil {
		t.Errorf("expected nil for empty tracks, got %v", matches)
	}

	matches = matcher.MatchTracks(context.Background(), []*domain.Track{}, "session", 1, nil)

	if matches != nil {
		t.Errorf("expected nil for empty slice, got %v", matches)
	}
}

func TestEvaluateMatch(t *testing.T) {
	tests := []struct {
		name           string
		sourceTrack    *domain.Track
		targetTrack    *domain.Track
		wantConfidence domain.MatchConfidence
	}{
		{
			name:           "exact match with official",
			sourceTrack:    mustTrack("Bohemian Rhapsody", "Queen"),
			targetTrack:    mustTrack("Queen - Bohemian Rhapsody (Official Video)", "Queen"),
			wantConfidence: domain.MatchConfidenceHigh,
		},
		{
			name:           "exact match without official",
			sourceTrack:    mustTrack("Bohemian Rhapsody", "Queen"),
			targetTrack:    mustTrack("Queen - Bohemian Rhapsody", "Queen Channel"),
			wantConfidence: domain.MatchConfidenceHigh,
		},
		{
			name:           "rejected - cover",
			sourceTrack:    mustTrack("Bohemian Rhapsody", "Queen"),
			targetTrack:    mustTrack("Bohemian Rhapsody Cover by Someone", "CoverChannel"),
			wantConfidence: domain.MatchConfidenceNone,
		},
		{
			name:           "rejected - live",
			sourceTrack:    mustTrack("Bohemian Rhapsody", "Queen"),
			targetTrack:    mustTrack("Bohemian Rhapsody Live at Wembley", "Queen"),
			wantConfidence: domain.MatchConfidenceNone,
		},
		{
			name:           "rejected - karaoke",
			sourceTrack:    mustTrack("Bohemian Rhapsody", "Queen"),
			targetTrack:    mustTrack("Bohemian Rhapsody Karaoke", "KaraokeChannel"),
			wantConfidence: domain.MatchConfidenceNone,
		},
		{
			name:           "partial match - title only",
			sourceTrack:    mustTrack("Bohemian Rhapsody", "Queen"),
			targetTrack:    mustTrack("Bohemian Rhapsody", "RandomChannel"),
			wantConfidence: domain.MatchConfidenceMedium,
		},
		{
			name:           "low confidence - no match",
			sourceTrack:    mustTrack("Bohemian Rhapsody", "Queen"),
			targetTrack:    mustTrack("Some Random Video", "RandomChannel"),
			wantConfidence: domain.MatchConfidenceLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence, _ := evaluateMatch(tt.sourceTrack, tt.targetTrack)
			if confidence != tt.wantConfidence {
				t.Errorf("evaluateMatch() confidence = %v, want %v", confidence, tt.wantConfidence)
			}
		})
	}
}

func mustTrack(name, artist string) *domain.Track {
	track, err := domain.NewTrack(name, artist, domain.PlatformSpotify, "test-id")
	if err != nil {
		panic(err)
	}
	return track
}
