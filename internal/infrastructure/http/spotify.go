package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type SpotifyClient interface {
	GetPlaylistTracks(ctx context.Context, playlistID, sessionID string) (*domain.Playlist, error)
}

type spotifyClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewSpotifyClient(cfg config.ServiceConfig) SpotifyClient {
	return &spotifyClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

type spotifyPlaylistResponse struct {
	Items  []spotifyTrackItem `json:"items"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type spotifyTrackItem struct {
	Track spotifyTrack `json:"track"`
}

type spotifyTrack struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Artists    []spotifyArtist `json:"artists"`
	Album      spotifyAlbum    `json:"album"`
	DurationMs int             `json:"durationMs"`
	ExternalID spotifyExternal `json:"externalIds"`
}

type spotifyArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type spotifyAlbum struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type spotifyExternal struct {
	ISRC string `json:"isrc"`
}

func (c *spotifyClient) GetPlaylistTracks(ctx context.Context, playlistID, sessionID string) (*domain.Playlist, error) {
	var allTracks []*domain.Track
	offset := 0
	limit := 50

	for {
		url := fmt.Sprintf("%s/api/spotify/v1/playlists/%s/tracks?limit=%d&offset=%d",
			c.baseURL, playlistID, limit, offset)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Cookie", "SESSIONID="+sessionID)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch playlist tracks: %w", err)
		}

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

		for _, item := range result.Items {
			track := toTrack(item.Track)
			if track != nil {
				allTracks = append(allTracks, track)
			}
		}

		if offset+limit >= result.Total {
			break
		}
		offset += limit
	}

	playlist, err := domain.NewPlaylist("", domain.PlatformSpotify, playlistID)
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

	artistName := ""
	if len(st.Artists) > 0 {
		artistName = st.Artists[0].Name
	}

	track, err := domain.NewTrack(st.Name, artistName, domain.PlatformSpotify, st.ID)
	if err != nil {
		return nil
	}

	track.WithAlbum(st.Album.Name).
		WithDuration(st.DurationMs).
		WithISRC(st.ExternalID.ISRC)

	return track
}
