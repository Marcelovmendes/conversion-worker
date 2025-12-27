package application

import (
	"context"
	"strings"
	"sync"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/http"
)

var excludeTerms = []string{"cover", "live", "karaoke", "remix", "tutorial", "reaction"}
var preferTerms = []string{"official", "audio", "video"}

type Matcher interface {
	MatchTracks(ctx context.Context, tracks []*domain.Track, sessionID string, concurrency int, onProgress func(processed, matched, failed int)) []*domain.TrackMatch
}

type matcher struct {
	youtubeClient http.YouTubeClient
}

func NewMatcher(youtubeClient http.YouTubeClient) Matcher {
	return &matcher{youtubeClient: youtubeClient}
}

func (m *matcher) MatchTracks(ctx context.Context, tracks []*domain.Track, sessionID string, concurrency int, onProgress func(processed, matched, failed int)) []*domain.TrackMatch {
	if len(tracks) == 0 {
		return nil
	}

	results := make(chan *domain.TrackMatch, len(tracks))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, track := range tracks {
		wg.Add(1)
		go func(t *domain.Track) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				results <- domain.NewFailedMatch(t, "context cancelled")
				return
			default:
			}

			match := m.matchTrack(ctx, t, sessionID)
			results <- match
		}(track)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var matches []*domain.TrackMatch
	processed, matched, failed := 0, 0, 0

	for match := range results {
		matches = append(matches, match)
		processed++

		if match.Confidence != domain.MatchConfidenceNone {
			matched++
		} else {
			failed++
		}

		if onProgress != nil {
			onProgress(processed, matched, failed)
		}
	}

	return matches
}

func (m *matcher) matchTrack(ctx context.Context, sourceTrack *domain.Track, sessionID string) *domain.TrackMatch {
	targetTrack, err := m.youtubeClient.SearchTrack(ctx, sourceTrack.Name, sourceTrack.Artist, sessionID)
	if err != nil {
		return domain.NewFailedMatch(sourceTrack, err.Error())
	}

	if targetTrack == nil {
		return domain.NewFailedMatch(sourceTrack, "no match found")
	}

	confidence, method := evaluateMatch(sourceTrack, targetTrack)
	if confidence == domain.MatchConfidenceNone {
		return domain.NewFailedMatch(sourceTrack, "match rejected by filter")
	}

	return domain.NewTrackMatch(sourceTrack, targetTrack, confidence, method)
}

func evaluateMatch(source *domain.Track, target *domain.Track) (domain.MatchConfidence, string) {
	titleLower := strings.ToLower(target.Name)

	for _, term := range excludeTerms {
		if strings.Contains(titleLower, term) {
			return domain.MatchConfidenceNone, ""
		}
	}

	artistLower := strings.ToLower(source.Artist)
	targetArtistLower := strings.ToLower(target.Artist)
	sourceTitleLower := strings.ToLower(source.Name)

	hasArtistMatch := strings.Contains(targetArtistLower, artistLower) ||
		strings.Contains(titleLower, artistLower)
	hasTitleMatch := strings.Contains(titleLower, sourceTitleLower)

	hasPreferredTerm := false
	for _, term := range preferTerms {
		if strings.Contains(titleLower, term) {
			hasPreferredTerm = true
			break
		}
	}

	if hasArtistMatch && hasTitleMatch {
		if hasPreferredTerm {
			return domain.MatchConfidenceHigh, "exact_match_official"
		}
		return domain.MatchConfidenceHigh, "exact_match"
	}

	if hasArtistMatch || hasTitleMatch {
		if hasPreferredTerm {
			return domain.MatchConfidenceMedium, "partial_match_official"
		}
		return domain.MatchConfidenceMedium, "partial_match"
	}

	return domain.MatchConfidenceLow, "first_result"
}
