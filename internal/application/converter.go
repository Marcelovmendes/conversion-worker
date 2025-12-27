package application

import (
	"context"
	"fmt"
	"log"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/http"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/postgres"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/redis"
)

type Converter interface {
	Convert(ctx context.Context, job *domain.ConversionJob) error
}

type converter struct {
	spotifyClient  http.SpotifyClient
	youtubeClient  http.YouTubeClient
	matcher        Matcher
	conversionRepo postgres.ConversionRepository
	logRepo        postgres.ConversionLogRepository
	statusStore    redis.StatusStore
	config         config.WorkerConfig
}

func NewConverter(
	spotifyClient http.SpotifyClient,
	youtubeClient http.YouTubeClient,
	matcher Matcher,
	conversionRepo postgres.ConversionRepository,
	logRepo postgres.ConversionLogRepository,
	statusStore redis.StatusStore,
	cfg config.WorkerConfig,
) Converter {
	return &converter{
		spotifyClient:  spotifyClient,
		youtubeClient:  youtubeClient,
		matcher:        matcher,
		conversionRepo: conversionRepo,
		logRepo:        logRepo,
		statusStore:    statusStore,
		config:         cfg,
	}
}

func (c *converter) Convert(ctx context.Context, job *domain.ConversionJob) error {
	conversion, err := domain.NewConversion(job)
	if err != nil {
		return fmt.Errorf("failed to create conversion: %w", err)
	}

	if err := c.conversionRepo.Create(ctx, conversion); err != nil {
		return fmt.Errorf("failed to persist conversion: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic during conversion %s: %v", conversion.ID, r)
			conversion.Fail(fmt.Sprintf("internal error: %v", r))
			c.saveState(ctx, conversion)
		}
	}()

	conversion.StartFetching()
	c.updateStatus(ctx, conversion)

	playlist, err := c.spotifyClient.GetPlaylistTracks(ctx, job.SourcePlaylistID, job.UserID)
	if err != nil {
		return c.handleError(ctx, conversion, "failed to fetch playlist", err)
	}

	c.logRepo.Create(ctx, domain.NewFetchPlaylistLog(conversion.ID, domain.LogStatusSuccess, ""))

	tracks := playlist.Tracks
	if len(job.SelectedTrackIDs) > 0 {
		tracks = filterTracks(tracks, job.SelectedTrackIDs)
	}

	conversion.StartMatching(len(tracks), playlist.Name)
	c.updateStatus(ctx, conversion)

	matches := c.matcher.MatchTracks(ctx, tracks, job.UserID, c.config.Concurrency, func(processed, matched, failed int) {
		conversion.UpdateProgress(processed, matched, failed)
		c.updateStatus(ctx, conversion)
	})

	var logs []*domain.ConversionLog
	var matchedVideoIDs []string

	for _, match := range matches {
		if match.Confidence != domain.MatchConfidenceNone {
			logs = append(logs, domain.NewMatchTrackLog(conversion.ID, match.SourceTrack, match.TargetTrack, domain.LogStatusSuccess))
			matchedVideoIDs = append(matchedVideoIDs, match.TargetTrack.PlatformID)
		} else {
			logs = append(logs, domain.NewMatchTrackErrorLog(conversion.ID, match.SourceTrack, match.Error))
		}
	}

	if err := c.logRepo.CreateBatch(ctx, logs); err != nil {
		log.Printf("failed to save match logs: %v", err)
	}

	if len(matchedVideoIDs) == 0 {
		return c.handleError(ctx, conversion, "no tracks matched", nil)
	}

	conversion.StartCreating()
	c.updateStatus(ctx, conversion)

	description := fmt.Sprintf("Converted from Spotify playlist: %s", playlist.Name)
	playlistID, playlistURL, err := c.youtubeClient.CreatePlaylist(ctx, job.TargetPlaylistName, description, job.UserID)
	if err != nil {
		c.logRepo.Create(ctx, domain.NewCreatePlaylistLog(conversion.ID, domain.LogStatusFailed, err.Error()))
		return c.handleError(ctx, conversion, "failed to create playlist", err)
	}

	c.logRepo.Create(ctx, domain.NewCreatePlaylistLog(conversion.ID, domain.LogStatusSuccess, ""))

	if err := c.youtubeClient.AddVideosToPlaylist(ctx, playlistID, matchedVideoIDs, job.UserID); err != nil {
		return c.handleError(ctx, conversion, "failed to add videos to playlist", err)
	}

	conversion.Complete(playlistID, playlistURL)
	c.saveState(ctx, conversion)

	log.Printf("conversion %s completed: %d/%d tracks matched, playlist: %s",
		conversion.ID, conversion.MatchedTracks, conversion.TotalTracks, playlistURL)

	return nil
}

func (c *converter) handleError(ctx context.Context, conversion *domain.Conversion, message string, err error) error {
	fullMessage := message
	if err != nil {
		fullMessage = fmt.Sprintf("%s: %v", message, err)
	}

	conversion.Fail(fullMessage)
	c.saveState(ctx, conversion)

	log.Printf("conversion %s failed: %s", conversion.ID, fullMessage)
	return fmt.Errorf("%s", fullMessage)
}

func (c *converter) updateStatus(ctx context.Context, conversion *domain.Conversion) {
	status := redis.NewStatusFromConversion(conversion)
	if err := c.statusStore.Set(ctx, status); err != nil {
		log.Printf("failed to update status in redis: %v", err)
	}
}

func (c *converter) saveState(ctx context.Context, conversion *domain.Conversion) {
	c.updateStatus(ctx, conversion)
	if err := c.conversionRepo.Update(ctx, conversion); err != nil {
		log.Printf("failed to update conversion in postgres: %v", err)
	}
}

func filterTracks(tracks []*domain.Track, selectedIDs []string) []*domain.Track {
	idSet := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		idSet[id] = true
	}

	var filtered []*domain.Track
	for _, track := range tracks {
		if idSet[track.PlatformID] {
			filtered = append(filtered, track)
		}
	}

	return filtered
}
