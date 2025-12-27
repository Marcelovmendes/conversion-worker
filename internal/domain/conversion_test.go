package domain

import (
	"testing"
	"time"
)

func TestNewConversion(t *testing.T) {
	job := &ConversionJob{
		JobID:              "job-123",
		UserID:             "user-456",
		SourcePlatform:     PlatformSpotify,
		TargetPlatform:     PlatformYouTube,
		SourcePlaylistID:   "playlist-789",
		TargetPlaylistName: "My Converted Playlist",
		CreatedAt:          time.Now(),
	}

	conversion, err := NewConversion(job)
	if err != nil {
		t.Fatalf("NewConversion() error: %v", err)
	}

	if conversion.ID != job.JobID {
		t.Errorf("conversion.ID = %q, want %q", conversion.ID, job.JobID)
	}
	if conversion.UserID != job.UserID {
		t.Errorf("conversion.UserID = %q, want %q", conversion.UserID, job.UserID)
	}
	if conversion.Status != ConversionStatusPending {
		t.Errorf("conversion.Status = %v, want %v", conversion.Status, ConversionStatusPending)
	}
}

func TestNewConversion_Validation(t *testing.T) {
	tests := []struct {
		name    string
		job     *ConversionJob
		wantErr bool
	}{
		{
			name:    "nil job",
			job:     nil,
			wantErr: true,
		},
		{
			name: "empty job ID",
			job: &ConversionJob{
				JobID:            "",
				UserID:           "user",
				SourcePlatform:   PlatformSpotify,
				TargetPlatform:   PlatformYouTube,
				SourcePlaylistID: "playlist",
			},
			wantErr: true,
		},
		{
			name: "empty user ID",
			job: &ConversionJob{
				JobID:            "job",
				UserID:           "",
				SourcePlatform:   PlatformSpotify,
				TargetPlatform:   PlatformYouTube,
				SourcePlaylistID: "playlist",
			},
			wantErr: true,
		},
		{
			name: "invalid source platform",
			job: &ConversionJob{
				JobID:            "job",
				UserID:           "user",
				SourcePlatform:   Platform("INVALID"),
				TargetPlatform:   PlatformYouTube,
				SourcePlaylistID: "playlist",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConversion(tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConversion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConversion_StateTransitions(t *testing.T) {
	job := NewConversionJob("user", PlatformSpotify, PlatformYouTube, "playlist", "My Playlist")
	conversion, _ := NewConversion(job)

	if conversion.Status != ConversionStatusPending {
		t.Fatalf("initial status should be PENDING")
	}

	conversion.StartFetching()
	if conversion.Status != ConversionStatusFetching {
		t.Errorf("after StartFetching, status = %v, want FETCHING", conversion.Status)
	}

	conversion.StartMatching(10, "Source Playlist")
	if conversion.Status != ConversionStatusMatching {
		t.Errorf("after StartMatching, status = %v, want MATCHING", conversion.Status)
	}
	if conversion.TotalTracks != 10 {
		t.Errorf("TotalTracks = %d, want 10", conversion.TotalTracks)
	}
	if conversion.SourcePlaylistName != "Source Playlist" {
		t.Errorf("SourcePlaylistName = %q, want %q", conversion.SourcePlaylistName, "Source Playlist")
	}

	conversion.UpdateProgress(5, 4, 1)
	if conversion.ProcessedTracks != 5 {
		t.Errorf("ProcessedTracks = %d, want 5", conversion.ProcessedTracks)
	}
	if conversion.MatchedTracks != 4 {
		t.Errorf("MatchedTracks = %d, want 4", conversion.MatchedTracks)
	}
	if conversion.FailedTracks != 1 {
		t.Errorf("FailedTracks = %d, want 1", conversion.FailedTracks)
	}

	conversion.StartCreating()
	if conversion.Status != ConversionStatusCreating {
		t.Errorf("after StartCreating, status = %v, want CREATING", conversion.Status)
	}

	conversion.Complete("yt-playlist-id", "https://youtube.com/playlist?list=xxx")
	if conversion.Status != ConversionStatusCompleted {
		t.Errorf("after Complete, status = %v, want COMPLETED", conversion.Status)
	}
	if conversion.TargetPlaylistID != "yt-playlist-id" {
		t.Errorf("TargetPlaylistID = %q, want %q", conversion.TargetPlaylistID, "yt-playlist-id")
	}
	if conversion.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}
}

func TestConversion_Fail(t *testing.T) {
	job := NewConversionJob("user", PlatformSpotify, PlatformYouTube, "playlist", "My Playlist")
	conversion, _ := NewConversion(job)

	conversion.StartFetching()
	conversion.Fail("something went wrong")

	if conversion.Status != ConversionStatusFailed {
		t.Errorf("after Fail, status = %v, want FAILED", conversion.Status)
	}
	if conversion.ErrorMessage != "something went wrong" {
		t.Errorf("ErrorMessage = %q, want %q", conversion.ErrorMessage, "something went wrong")
	}
	if conversion.CompletedAt == nil {
		t.Error("CompletedAt should not be nil after failure")
	}
}

func TestConversion_Progress(t *testing.T) {
	job := NewConversionJob("user", PlatformSpotify, PlatformYouTube, "playlist", "My Playlist")
	conversion, _ := NewConversion(job)

	if conversion.Progress() != 0 {
		t.Errorf("initial Progress() = %d, want 0", conversion.Progress())
	}

	conversion.TotalTracks = 100
	conversion.ProcessedTracks = 50
	if conversion.Progress() != 50 {
		t.Errorf("Progress() = %d, want 50", conversion.Progress())
	}

	conversion.ProcessedTracks = 100
	if conversion.Progress() != 100 {
		t.Errorf("Progress() = %d, want 100", conversion.Progress())
	}
}

func TestConversionStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   ConversionStatus
		terminal bool
	}{
		{ConversionStatusPending, false},
		{ConversionStatusFetching, false},
		{ConversionStatusMatching, false},
		{ConversionStatusCreating, false},
		{ConversionStatusCompleted, true},
		{ConversionStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.terminal)
			}
		})
	}
}
