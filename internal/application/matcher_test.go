package application

import (
	"context"
	"errors"
	"testing"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type mockYouTubeClient struct {
	isrcResults  map[string]*domain.Track
	trackResults map[string][]*domain.Track
	searchError  error
}

func (m *mockYouTubeClient) SearchByISRC(ctx context.Context, isrc, sessionID string) (*domain.Track, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.isrcResults[isrc], nil
}

func (m *mockYouTubeClient) SearchTrack(ctx context.Context, track, artist, sessionID string) ([]*domain.Track, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	key := track + "|" + artist
	return m.trackResults[key], nil
}

func (m *mockYouTubeClient) CreatePlaylist(ctx context.Context, name, description, sessionID string) (string, string, error) {
	return "playlist-id", "https://youtube.com/playlist?list=xxx", nil
}

func (m *mockYouTubeClient) AddVideosToPlaylist(ctx context.Context, playlistID string, videoIDs []string, sessionID string) error {
	return nil
}

func TestMatcher_ISRC(t *testing.T) {
	ytTrack, _ := domain.NewTrack("Bohemian Rhapsody (Official Video)", "Queen", domain.PlatformYouTube, "yt1")

	mockClient := &mockYouTubeClient{
		isrcResults: map[string]*domain.Track{
			"GBUM71029604": ytTrack,
		},
		trackResults: map[string][]*domain.Track{},
	}

	sourceTrack, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")
	sourceTrack.WithISRC("GBUM71029604")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceHigh {
		t.Errorf("expected HIGH confidence, got %v", matches[0].Confidence)
	}

	if matches[0].MatchMethod != "isrc" {
		t.Errorf("expected method 'isrc', got %s", matches[0].MatchMethod)
	}
}

func TestMatcher_ExactMatch(t *testing.T) {
	ytTrack, _ := domain.NewTrack("Queen - Bohemian Rhapsody (Official Video)", "Queen", domain.PlatformYouTube, "yt1")

	mockClient := &mockYouTubeClient{
		isrcResults: map[string]*domain.Track{},
		trackResults: map[string][]*domain.Track{
			"Bohemian Rhapsody|Queen": {ytTrack},
		},
	}

	sourceTrack, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceHigh {
		t.Errorf("expected HIGH confidence, got %v", matches[0].Confidence)
	}

	if matches[0].MatchMethod != "exact_match" {
		t.Errorf("expected method 'exact_match', got %s", matches[0].MatchMethod)
	}
}

func TestMatcher_PartialMatch(t *testing.T) {
	ytTrack, _ := domain.NewTrack("Bohemian Rhapsody Audio", "SomeChannel", domain.PlatformYouTube, "yt1")

	mockClient := &mockYouTubeClient{
		isrcResults: map[string]*domain.Track{},
		trackResults: map[string][]*domain.Track{
			"Bohemian Rhapsody|Queen": {ytTrack},
		},
	}

	sourceTrack, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceMedium {
		t.Errorf("expected MEDIUM confidence, got %v", matches[0].Confidence)
	}

	if matches[0].MatchMethod != "partial_match" {
		t.Errorf("expected method 'partial_match', got %s", matches[0].MatchMethod)
	}
}

func TestMatcher_LowConfidence(t *testing.T) {
	ytTrack, _ := domain.NewTrack("Some Music Video", "RandomChannel", domain.PlatformYouTube, "yt1")

	mockClient := &mockYouTubeClient{
		isrcResults: map[string]*domain.Track{},
		trackResults: map[string][]*domain.Track{
			"Bohemian Rhapsody|Queen": {ytTrack},
		},
	}

	sourceTrack, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceLow {
		t.Errorf("expected LOW confidence, got %v", matches[0].Confidence)
	}

	if matches[0].MatchMethod != "music_search" {
		t.Errorf("expected method 'music_search', got %s", matches[0].MatchMethod)
	}
}

func TestMatcher_NoResults(t *testing.T) {
	mockClient := &mockYouTubeClient{
		isrcResults:  map[string]*domain.Track{},
		trackResults: map[string][]*domain.Track{},
	}

	sourceTrack, _ := domain.NewTrack("Unknown Song", "Unknown Artist", domain.PlatformSpotify, "sp1")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceNone {
		t.Errorf("expected NONE confidence, got %v", matches[0].Confidence)
	}

	if matches[0].Error == "" {
		t.Error("expected error message for failed match")
	}
}

func TestMatcher_ExcludesCovers(t *testing.T) {
	ytCover, _ := domain.NewTrack("Bohemian Rhapsody Cover", "CoverChannel", domain.PlatformYouTube, "yt1")

	mockClient := &mockYouTubeClient{
		isrcResults: map[string]*domain.Track{},
		trackResults: map[string][]*domain.Track{
			"Bohemian Rhapsody|Queen": {ytCover},
		},
	}

	sourceTrack, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceNone {
		t.Errorf("expected NONE confidence (cover excluded), got %v", matches[0].Confidence)
	}
}

func TestMatcher_ExcludesLive(t *testing.T) {
	ytLive, _ := domain.NewTrack("Bohemian Rhapsody Live at Wembley", "Queen", domain.PlatformYouTube, "yt1")

	mockClient := &mockYouTubeClient{
		isrcResults: map[string]*domain.Track{},
		trackResults: map[string][]*domain.Track{
			"Bohemian Rhapsody|Queen": {ytLive},
		},
	}

	sourceTrack, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if matches[0].Confidence != domain.MatchConfidenceNone {
		t.Errorf("expected NONE confidence (live excluded), got %v", matches[0].Confidence)
	}
}

func TestMatcher_EmptyTracks(t *testing.T) {
	mockClient := &mockYouTubeClient{}
	matcher := NewMatcher(mockClient)

	matches := matcher.MatchTracks(context.Background(), nil, "session", 1, nil)
	if matches != nil {
		t.Errorf("expected nil for nil tracks, got %v", matches)
	}

	matches = matcher.MatchTracks(context.Background(), []*domain.Track{}, "session", 1, nil)
	if matches != nil {
		t.Errorf("expected nil for empty slice, got %v", matches)
	}
}

func TestMatcher_ProgressCallback(t *testing.T) {
	ytTrack, _ := domain.NewTrack("Track 1", "Artist", domain.PlatformYouTube, "yt1")

	mockClient := &mockYouTubeClient{
		isrcResults: map[string]*domain.Track{},
		trackResults: map[string][]*domain.Track{
			"Track 1|Artist": {ytTrack},
			"Track 2|Artist": {ytTrack},
		},
	}

	track1, _ := domain.NewTrack("Track 1", "Artist", domain.PlatformSpotify, "sp1")
	track2, _ := domain.NewTrack("Track 2", "Artist", domain.PlatformSpotify, "sp2")

	matcher := NewMatcher(mockClient)

	var progressCalls int
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{track1, track2}, "session", 2, func(processed, matched, failed int) {
		progressCalls++
	})

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	if progressCalls != 2 {
		t.Errorf("expected 2 progress calls, got %d", progressCalls)
	}
}

func TestMatcher_SearchError(t *testing.T) {
	mockClient := &mockYouTubeClient{
		searchError: errors.New("network error"),
	}

	sourceTrack, _ := domain.NewTrack("Test Track", "Test Artist", domain.PlatformSpotify, "sp1")

	matcher := NewMatcher(mockClient)
	matches := matcher.MatchTracks(context.Background(), []*domain.Track{sourceTrack}, "session", 1, nil)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].Confidence != domain.MatchConfidenceNone {
		t.Errorf("expected NONE confidence on error, got %v", matches[0].Confidence)
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		title    string
		excluded bool
	}{
		{"Bohemian Rhapsody Official Video", false},
		{"Bohemian Rhapsody Cover", true},
		{"Bohemian Rhapsody COVER by Someone", true},
		{"Bohemian Rhapsody Live", true},
		{"Bohemian Rhapsody Karaoke", true},
		{"Bohemian Rhapsody Remix", true},
		{"Bohemian Rhapsody Tutorial", true},
		{"Bohemian Rhapsody Reaction", true},
		{"Queen - Bohemian Rhapsody", false},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			if isExcluded(tt.title) != tt.excluded {
				t.Errorf("isExcluded(%q) = %v, want %v", tt.title, !tt.excluded, tt.excluded)
			}
		})
	}
}

func TestHasArtistMatch(t *testing.T) {
	source, _ := domain.NewTrack("Song", "Queen", domain.PlatformSpotify, "sp1")

	tests := []struct {
		name     string
		target   *domain.Track
		expected bool
	}{
		{
			name:     "artist in channel name",
			target:   mustTrack("Some Song", "Queen Official"),
			expected: true,
		},
		{
			name:     "artist in title",
			target:   mustTrack("Queen - Some Song", "Random Channel"),
			expected: true,
		},
		{
			name:     "no match",
			target:   mustTrack("Some Song", "Random Channel"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if hasArtistMatch(source, tt.target) != tt.expected {
				t.Errorf("hasArtistMatch() = %v, want %v", !tt.expected, tt.expected)
			}
		})
	}
}

func TestHasTitleMatch(t *testing.T) {
	source, _ := domain.NewTrack("Bohemian Rhapsody", "Queen", domain.PlatformSpotify, "sp1")

	tests := []struct {
		name     string
		target   *domain.Track
		expected bool
	}{
		{
			name:     "title contained",
			target:   mustTrack("Queen - Bohemian Rhapsody (Official)", "Queen"),
			expected: true,
		},
		{
			name:     "exact title",
			target:   mustTrack("Bohemian Rhapsody", "Queen"),
			expected: true,
		},
		{
			name:     "no match",
			target:   mustTrack("Another Song", "Queen"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if hasTitleMatch(source, tt.target) != tt.expected {
				t.Errorf("hasTitleMatch() = %v, want %v", !tt.expected, tt.expected)
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
