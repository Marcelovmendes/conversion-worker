package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type YouTubeClient interface {
	SearchTrack(ctx context.Context, trackName, artistName, sessionID string) (*domain.Track, error)
	CreatePlaylist(ctx context.Context, name, description, sessionID string) (playlistID string, playlistURL string, err error)
	AddVideosToPlaylist(ctx context.Context, playlistID string, videoIDs []string, sessionID string) error
}

type youtubeClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewYouTubeClient(cfg config.ServiceConfig) YouTubeClient {
	return &youtubeClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

type youtubeSearchResponse struct {
	Items []youtubeVideo `json:"items"`
}

type youtubeVideo struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ChannelTitle string `json:"channelTitle"`
	Duration     int    `json:"duration"`
}

type createPlaylistRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type createPlaylistResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type addVideosRequest struct {
	VideoIDs []string `json:"videoIds"`
}

func (c *youtubeClient) SearchTrack(ctx context.Context, trackName, artistName, sessionID string) (*domain.Track, error) {
	query := url.QueryEscape(fmt.Sprintf("%s %s", artistName, trackName))
	searchURL := fmt.Sprintf("%s/api/youtube/v1/search/music?q=%s", c.baseURL, query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Cookie", "YOUTUBE_SESSION="+sessionID)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search track: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("youtube service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result youtubeSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	video := result.Items[0]
	track, err := domain.NewTrack(video.Title, video.ChannelTitle, domain.PlatformYouTube, video.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create track: %w", err)
	}

	track.WithDuration(video.Duration)
	return track, nil
}

func (c *youtubeClient) CreatePlaylist(ctx context.Context, name, description, sessionID string) (string, string, error) {
	createURL := fmt.Sprintf("%s/api/youtube/v1/playlists", c.baseURL)

	reqBody := createPlaylistRequest{
		Title:       name,
		Description: description,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Cookie", "YOUTUBE_SESSION="+sessionID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to create playlist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("youtube service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result createPlaylistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, result.URL, nil
}

func (c *youtubeClient) AddVideosToPlaylist(ctx context.Context, playlistID string, videoIDs []string, sessionID string) error {
	if len(videoIDs) == 0 {
		return nil
	}

	addURL := fmt.Sprintf("%s/api/youtube/v1/playlists/%s/videos", c.baseURL, playlistID)

	reqBody := addVideosRequest{VideoIDs: videoIDs}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Cookie", "YOUTUBE_SESSION="+sessionID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add videos to playlist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("youtube service returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
