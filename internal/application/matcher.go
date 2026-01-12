package application

import (
	"context"
	"log"
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
	if match := m.tryISRCSearch(ctx, sourceTrack, sessionID); match != nil {
		return match
	}

	if match := m.tryMusicSearch(ctx, sourceTrack, sessionID); match != nil {
		return match
	}

	return domain.NewFailedMatch(sourceTrack, "no match found")
}

func (m *matcher) tryISRCSearch(ctx context.Context, sourceTrack *domain.Track, sessionID string) *domain.TrackMatch {
	if sourceTrack.ISRC == "" {
		return nil
	}

	targetTrack, err := m.youtubeClient.SearchByISRC(ctx, sourceTrack.ISRC, sessionID)
	if err != nil {
		log.Printf("[DEBUG] ISRC search failed for %q: %v", sourceTrack.Name, err)
		return nil
	}
	if targetTrack == nil {
		return nil
	}

	return domain.NewTrackMatch(sourceTrack, targetTrack, domain.MatchConfidenceHigh, "isrc")
}

func (m *matcher) tryMusicSearch(ctx context.Context, sourceTrack *domain.Track, sessionID string) *domain.TrackMatch {
	log.Printf("[DEBUG] searching YouTube for track=%q artist=%q", sourceTrack.Name, sourceTrack.Artist)

	tracks, err := m.youtubeClient.SearchTrack(ctx, sourceTrack.Name, sourceTrack.Artist, sessionID)
	if err != nil {
		log.Printf("[DEBUG] music search failed for %q - %q: %v", sourceTrack.Artist, sourceTrack.Name, err)
		return nil
	}
	if len(tracks) == 0 {
		log.Printf("[DEBUG] music search returned 0 results for %q - %q", sourceTrack.Artist, sourceTrack.Name)
		return nil
	}

	log.Printf("[DEBUG] music search returned %d results", len(tracks))

	for _, targetTrack := range tracks {
		if isExcluded(targetTrack.Name) {
			continue
		}

		confidence := domain.MatchConfidenceLow
		method := "music_search"

		if hasArtistMatch(sourceTrack, targetTrack) && hasTitleMatch(sourceTrack, targetTrack) {
			confidence = domain.MatchConfidenceHigh
			method = "exact_match"
		} else if hasArtistMatch(sourceTrack, targetTrack) || hasTitleMatch(sourceTrack, targetTrack) {
			confidence = domain.MatchConfidenceMedium
			method = "partial_match"
		}

		return domain.NewTrackMatch(sourceTrack, targetTrack, confidence, method)
	}

	return nil
}

func isExcluded(title string) bool {
	titleLower := strings.ToLower(title)
	for _, term := range excludeTerms {
		if strings.Contains(titleLower, term) {
			return true
		}
	}
	return false
}

func hasArtistMatch(source, target *domain.Track) bool {
	artistLower := strings.ToLower(source.Artist)
	targetArtistLower := strings.ToLower(target.Artist)
	titleLower := strings.ToLower(target.Name)

	return strings.Contains(targetArtistLower, artistLower) ||
		strings.Contains(titleLower, artistLower)
}

func hasTitleMatch(source, target *domain.Track) bool {
	sourceTitleLower := strings.ToLower(source.Name)
	targetTitleLower := strings.ToLower(target.Name)

	return strings.Contains(targetTitleLower, sourceTitleLower)
}

func hasPreferredTerm(title string) bool {
	titleLower := strings.ToLower(title)
	for _, term := range preferTerms {
		if strings.Contains(titleLower, term) {
			return true
		}
	}
	return false
}
