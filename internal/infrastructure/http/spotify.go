package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/redis"
)

type SpotifyClient interface {
	GetPlaylistTracks(ctx context.Context, playlistID, sessionID string) (*domain.Playlist, error)
}

type spotifyClient struct {
	baseURL      string
	httpClient   *http.Client
	sessionStore redis.SessionStore
}

func NewSpotifyClient(cfg config.ServiceConfig, sessionStore redis.SessionStore) SpotifyClient {
	return &spotifyClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		sessionStore: sessionStore,
	}
}

type spotifyPlaylistResponse struct {
	Items  []spotifyTrack `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type spotifyTrack struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int    `json:"durationMs"`
}

func (c *spotifyClient) GetPlaylistTracks(ctx context.Context, playlistID, sessionID string) (*domain.Playlist, error) {
	log.Printf("[DEBUG] fetching Spotify token for session: %s", sessionID)

	token, err := c.sessionStore.GetSpotifyToken(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get spotify token: %w", err)
	}

	log.Printf("[DEBUG] got Spotify token, expires at: %s", token.ExpiresAt)

	var allTracks []*domain.Track
	var playlistName string
	offset := 0
	limit := 50

	for {
		url := fmt.Sprintf("%s/internal/playlists/%s/tracks?limit=%d&offset=%d",
			c.baseURL, playlistID, limit, offset)

		log.Printf("[DEBUG] Spotify request URL: %s", url)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch playlist tracks: %w", err)
		}

		log.Printf("[DEBUG] Spotify response status: %d", resp.StatusCode)

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("spotify service returned status %d: %s", resp.StatusCode, string(body))
		}

		var result spotifyPlaylistResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		log.Printf("[DEBUG] Spotify response: items=%d, total=%d", len(result.Items), result.Total)

		for _, item := range result.Items {
			track := toTrack(item)
			if track != nil {
				allTracks = append(allTracks, track)
			}
		}

		if offset+limit >= result.Total {
			break
		}
		offset += limit
	}

	if playlistName == "" {
		playlistName = "Playlist"
	}

	playlist, err := domain.NewPlaylist(playlistName, domain.PlatformSpotify, playlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	playlist.AddTracks(allTracks)
	return playlist, nil
}

func toTrack(st spotifyTrack) *domain.Track {
	if st.ID == "" || st.Name == "" {
		return nil
	}

	track, err := domain.NewTrack(st.Name, st.Artist, domain.PlatformSpotify, st.ID)
	if err != nil {
		return nil
	}

	track.WithAlbum(st.Album).WithDuration(st.DurationMs)

	return track
}
