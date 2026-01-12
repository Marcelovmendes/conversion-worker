package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/redis"
)

type YouTubeClient interface {
	SearchByISRC(ctx context.Context, isrc, sessionID string) (*domain.Track, error)
	SearchTrack(ctx context.Context, track, artist, sessionID string) ([]*domain.Track, error)
	CreatePlaylist(ctx context.Context, name, description, sessionID string) (playlistID string, playlistURL string, err error)
	AddVideosToPlaylist(ctx context.Context, playlistID string, videoIDs []string, sessionID string) error
}

type youtubeClient struct {
	baseURL      string
	httpClient   *http.Client
	sessionStore redis.SessionStore
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

func NewYouTubeClient(cfg config.ServiceConfig, sessionStore redis.SessionStore) YouTubeClient {
	return &youtubeClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		sessionStore: sessionStore,
	}
}

func (c *youtubeClient) getAuthHeader(ctx context.Context, spotifySessionID string) (string, error) {
	token, err := c.sessionStore.GetYouTubeToken(ctx, spotifySessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get youtube token: %w", err)
	}
	accessToken := sanitizeToken(token.AccessToken)
	log.Printf("[DEBUG] YouTube token length: %d", len(accessToken))
	return "Bearer " + accessToken, nil
}

func sanitizeToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.ReplaceAll(token, "\n", "")
	token = strings.ReplaceAll(token, "\r", "")
	return token
}

type youtubeSearchResponse struct {
	VideoID        string  `json:"videoId"`
	Title          string  `json:"title"`
	ChannelTitle   string  `json:"channelTitle"`
	Description    string  `json:"description"`
	ThumbnailURL   string  `json:"thumbnailUrl"`
	RelevanceScore float64 `json:"relevanceScore"`
}

func (c *youtubeClient) SearchByISRC(ctx context.Context, isrc, sessionID string) (*domain.Track, error) {
	if isrc == "" {
		return nil, nil
	}

	authHeader, err := c.getAuthHeader(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	searchURL := fmt.Sprintf("%s/v1/search/track?isrc=%s", c.baseURL, url.QueryEscape(isrc))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search by ISRC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("youtube service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result youtubeSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.VideoID == "" {
		return nil, nil
	}

	return c.responseToTrack(result)
}

func (c *youtubeClient) SearchTrack(ctx context.Context, track, artist, sessionID string) ([]*domain.Track, error) {
	authHeader, err := c.getAuthHeader(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	searchURL := fmt.Sprintf("%s/v1/search/music?track=%s&artist=%s",
		c.baseURL, url.QueryEscape(track), url.QueryEscape(artist))

	log.Printf("[DEBUG] YouTube search URL: %s", searchURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search track: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] YouTube search response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("youtube service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result youtubeSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.VideoID == "" {
		return nil, nil
	}

	domainTrack, err := c.responseToTrack(result)
	if err != nil {
		return nil, err
	}

	return []*domain.Track{domainTrack}, nil
}

func (c *youtubeClient) responseToTrack(resp youtubeSearchResponse) (*domain.Track, error) {
	track, err := domain.NewTrack(resp.Title, resp.ChannelTitle, domain.PlatformYouTube, resp.VideoID)
	if err != nil {
		return nil, fmt.Errorf("failed to create track: %w", err)
	}
	return track, nil
}

func (c *youtubeClient) CreatePlaylist(ctx context.Context, name, description, sessionID string) (string, string, error) {
	authHeader, err := c.getAuthHeader(ctx, sessionID)
	if err != nil {
		return "", "", err
	}

	createURL := fmt.Sprintf("%s/v1/playlists", c.baseURL)

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

	req.Header.Set("Authorization", authHeader)
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

	authHeader, err := c.getAuthHeader(ctx, sessionID)
	if err != nil {
		return err
	}

	addURL := fmt.Sprintf("%s/v1/playlists/%s/videos", c.baseURL, playlistID)

	reqBody := addVideosRequest{VideoIDs: videoIDs}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", authHeader)
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
