package domain

import "testing"

func TestNewTrack(t *testing.T) {
	tests := []struct {
		name       string
		trackName  string
		artist     string
		platform   Platform
		platformID string
		wantErr    bool
	}{
		{
			name:       "valid track",
			trackName:  "Bohemian Rhapsody",
			artist:     "Queen",
			platform:   PlatformSpotify,
			platformID: "spotify123",
			wantErr:    false,
		},
		{
			name:       "empty name",
			trackName:  "",
			artist:     "Queen",
			platform:   PlatformSpotify,
			platformID: "spotify123",
			wantErr:    true,
		},
		{
			name:       "empty artist",
			trackName:  "Bohemian Rhapsody",
			artist:     "",
			platform:   PlatformSpotify,
			platformID: "spotify123",
			wantErr:    true,
		},
		{
			name:       "invalid platform",
			trackName:  "Bohemian Rhapsody",
			artist:     "Queen",
			platform:   Platform("INVALID"),
			platformID: "spotify123",
			wantErr:    true,
		},
		{
			name:       "empty platform ID",
			trackName:  "Bohemian Rhapsody",
			artist:     "Queen",
			platform:   PlatformSpotify,
			platformID: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track, err := NewTrack(tt.trackName, tt.artist, tt.platform, tt.platformID)

			if tt.wantErr {
				if err == nil {
					t.Error("NewTrack() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewTrack() unexpected error: %v", err)
				return
			}

			if track.Name != tt.trackName {
				t.Errorf("track.Name = %q, want %q", track.Name, tt.trackName)
			}
			if track.Artist != tt.artist {
				t.Errorf("track.Artist = %q, want %q", track.Artist, tt.artist)
			}
			if track.Platform != tt.platform {
				t.Errorf("track.Platform = %v, want %v", track.Platform, tt.platform)
			}
			if track.PlatformID != tt.platformID {
				t.Errorf("track.PlatformID = %q, want %q", track.PlatformID, tt.platformID)
			}
			if track.ID == "" {
				t.Error("track.ID should not be empty")
			}
		})
	}
}

func TestTrack_WithMethods(t *testing.T) {
	track, err := NewTrack("Test Song", "Test Artist", PlatformSpotify, "test123")
	if err != nil {
		t.Fatalf("NewTrack() error: %v", err)
	}

	track.WithAlbum("Test Album").
		WithDuration(180000).
		WithISRC("USRC12345678")

	if track.Album != "Test Album" {
		t.Errorf("track.Album = %q, want %q", track.Album, "Test Album")
	}
	if track.DurationMs != 180000 {
		t.Errorf("track.DurationMs = %d, want %d", track.DurationMs, 180000)
	}
	if track.ISRC != "USRC12345678" {
		t.Errorf("track.ISRC = %q, want %q", track.ISRC, "USRC12345678")
	}
}

func TestNewTrackMatch(t *testing.T) {
	source, _ := NewTrack("Source Song", "Artist", PlatformSpotify, "src123")
	target, _ := NewTrack("Target Song", "Artist", PlatformYouTube, "tgt123")

	match := NewTrackMatch(source, target, MatchConfidenceHigh, "exact_match")

	if match.SourceTrack != source {
		t.Error("match.SourceTrack should be source")
	}
	if match.TargetTrack != target {
		t.Error("match.TargetTrack should be target")
	}
	if match.Confidence != MatchConfidenceHigh {
		t.Errorf("match.Confidence = %v, want %v", match.Confidence, MatchConfidenceHigh)
	}
	if match.MatchMethod != "exact_match" {
		t.Errorf("match.MatchMethod = %q, want %q", match.MatchMethod, "exact_match")
	}
}

func TestNewFailedMatch(t *testing.T) {
	source, _ := NewTrack("Source Song", "Artist", PlatformSpotify, "src123")

	match := NewFailedMatch(source, "no match found")

	if match.SourceTrack != source {
		t.Error("match.SourceTrack should be source")
	}
	if match.TargetTrack != nil {
		t.Error("match.TargetTrack should be nil")
	}
	if match.Confidence != MatchConfidenceNone {
		t.Errorf("match.Confidence = %v, want %v", match.Confidence, MatchConfidenceNone)
	}
	if match.Error != "no match found" {
		t.Errorf("match.Error = %q, want %q", match.Error, "no match found")
	}
}
