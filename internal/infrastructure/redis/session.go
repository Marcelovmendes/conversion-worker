package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	springSessionPrefix   = "spring:session:sessions:"
	accessTokenAttr       = "sessionAttr:spotifyAccessToken"
	refreshTokenAttr      = "sessionAttr:spotifyRefreshToken"
	tokenExpiryAttr       = "sessionAttr:spotifyTokenExpiry"
	youtubeSessionIdAttr  = "sessionAttr:youtubeSessionId"

	youtubeAccessTokenAttr  = "sessionAttr:youtubeAccessToken"
	youtubeRefreshTokenAttr = "sessionAttr:youtubeRefreshToken"
	youtubeTokenExpiryAttr  = "sessionAttr:youtubeTokenExpiry"
)

type SpotifyToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (t *SpotifyToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

type YouTubeToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (t *YouTubeToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

type SessionStore interface {
	GetSpotifyToken(ctx context.Context, sessionID string) (*SpotifyToken, error)
	GetYouTubeToken(ctx context.Context, spotifySessionID string) (*YouTubeToken, error)
}

type sessionStore struct {
	rdb *redis.Client
}

func NewSessionStore(client Client) SessionStore {
	return &sessionStore{rdb: client.GetRDB()}
}

func (s *sessionStore) GetSpotifyToken(ctx context.Context, sessionID string) (*SpotifyToken, error) {
	key := springSessionPrefix + sessionID

	results, err := s.rdb.HMGet(ctx, key, accessTokenAttr, refreshTokenAttr, tokenExpiryAttr).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session attributes: %w", err)
	}

	if results[0] == nil || results[1] == nil || results[2] == nil {
		return nil, fmt.Errorf("session not found or missing token attributes")
	}

	accessToken, err := parseJSONString(results[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}

	refreshToken, err := parseJSONString(results[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	expiryMillis, err := parseJSONInt64(results[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse token expiry: %w", err)
	}

	expiresAt := time.UnixMilli(expiryMillis)

	token := &SpotifyToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}

	if token.IsExpired() {
		return nil, fmt.Errorf("token expired")
	}

	return token, nil
}

func (s *sessionStore) GetYouTubeToken(ctx context.Context, spotifySessionID string) (*YouTubeToken, error) {
	spotifyKey := springSessionPrefix + spotifySessionID

	youtubeSessionResult, err := s.rdb.HGet(ctx, spotifyKey, youtubeSessionIdAttr).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get youtube session id from spotify session: %w", err)
	}

	log.Printf("[DEBUG] Raw youtubeSessionResult: %q", youtubeSessionResult)

	youtubeSessionID, err := parseJSONString(youtubeSessionResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse youtube session id: %w", err)
	}

	log.Printf("[DEBUG] Parsed youtubeSessionID: %s", youtubeSessionID)

	youtubeKey := springSessionPrefix + youtubeSessionID

	results, err := s.rdb.HMGet(ctx, youtubeKey, youtubeAccessTokenAttr, youtubeRefreshTokenAttr, youtubeTokenExpiryAttr).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get youtube session attributes: %w", err)
	}

	if results[0] == nil {
		return nil, fmt.Errorf("youtube session not found or missing token attributes")
	}

	log.Printf("[DEBUG] Raw accessToken from Redis: %q", results[0])

	accessToken, err := parseJSONString(results[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse youtube access token: %w", err)
	}

	log.Printf("[DEBUG] Parsed accessToken length: %d, first 20 chars: %s", len(accessToken), accessToken[:min(20, len(accessToken))])

	var refreshToken string
	if results[1] != nil {
		refreshToken, _ = parseJSONString(results[1])
	}

	var expiresAt time.Time
	if results[2] != nil {
		expiryMillis, err := parseJSONInt64(results[2])
		if err == nil {
			expiresAt = time.UnixMilli(expiryMillis)
		}
	}

	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(time.Hour)
	}

	token := &YouTubeToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}

	if token.IsExpired() {
		return nil, fmt.Errorf("youtube token expired")
	}

	return token, nil
}

func parseJSONString(v interface{}) (string, error) {
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", v)
	}

	var result string
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return str, nil
	}
	return result, nil
}

func parseJSONInt64(v interface{}) (int64, error) {
	str, ok := v.(string)
	if !ok {
		return 0, fmt.Errorf("expected string, got %T", v)
	}

	var result int64
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return 0, fmt.Errorf("failed to parse int64: %w", err)
	}
	return result, nil
}
