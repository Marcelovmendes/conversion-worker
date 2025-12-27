package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type ConversionRepository interface {
	Create(ctx context.Context, c *domain.Conversion) error
	FindByID(ctx context.Context, id string) (*domain.Conversion, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversion, error)
	Update(ctx context.Context, c *domain.Conversion) error
}

type conversionRepository struct {
	pool *pgxpool.Pool
}

func NewConversionRepository(client Client) ConversionRepository {
	return &conversionRepository{pool: client.GetPool()}
}

func (r *conversionRepository) Create(ctx context.Context, c *domain.Conversion) error {
	query := `
		INSERT INTO conversions (
			id, user_id, source_platform, target_platform,
			source_playlist_id, source_playlist_name,
			target_playlist_id, target_playlist_url, target_playlist_name,
			status, total_tracks, processed_tracks, matched_tracks, failed_tracks,
			error_message, created_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`

	_, err := r.pool.Exec(ctx, query,
		c.ID,
		c.UserID,
		c.SourcePlatform,
		c.TargetPlatform,
		c.SourcePlaylistID,
		nullableString(c.SourcePlaylistName),
		nullableString(c.TargetPlaylistID),
		nullableString(c.TargetPlaylistURL),
		nullableString(c.TargetPlaylistName),
		c.Status,
		c.TotalTracks,
		c.ProcessedTracks,
		c.MatchedTracks,
		c.FailedTracks,
		nullableString(c.ErrorMessage),
		c.CreatedAt,
		c.UpdatedAt,
		c.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create conversion: %w", err)
	}

	return nil
}

func (r *conversionRepository) FindByID(ctx context.Context, id string) (*domain.Conversion, error) {
	query := `
		SELECT
			id, user_id, source_platform, target_platform,
			source_playlist_id, source_playlist_name,
			target_playlist_id, target_playlist_url, target_playlist_name,
			status, total_tracks, processed_tracks, matched_tracks, failed_tracks,
			error_message, created_at, updated_at, completed_at
		FROM conversions
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	return scanConversion(row)
}

func (r *conversionRepository) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversion, error) {
	query := `
		SELECT
			id, user_id, source_platform, target_platform,
			source_playlist_id, source_playlist_name,
			target_playlist_id, target_playlist_url, target_playlist_name,
			status, total_tracks, processed_tracks, matched_tracks, failed_tracks,
			error_message, created_at, updated_at, completed_at
		FROM conversions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversions: %w", err)
	}
	defer rows.Close()

	var conversions []*domain.Conversion
	for rows.Next() {
		c, err := scanConversionFromRows(rows)
		if err != nil {
			return nil, err
		}
		conversions = append(conversions, c)
	}

	return conversions, nil
}

func (r *conversionRepository) Update(ctx context.Context, c *domain.Conversion) error {
	query := `
		UPDATE conversions SET
			source_playlist_name = $2,
			target_playlist_id = $3,
			target_playlist_url = $4,
			target_playlist_name = $5,
			status = $6,
			total_tracks = $7,
			processed_tracks = $8,
			matched_tracks = $9,
			failed_tracks = $10,
			error_message = $11,
			updated_at = $12,
			completed_at = $13
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		c.ID,
		nullableString(c.SourcePlaylistName),
		nullableString(c.TargetPlaylistID),
		nullableString(c.TargetPlaylistURL),
		nullableString(c.TargetPlaylistName),
		c.Status,
		c.TotalTracks,
		c.ProcessedTracks,
		c.MatchedTracks,
		c.FailedTracks,
		nullableString(c.ErrorMessage),
		c.UpdatedAt,
		c.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update conversion: %w", err)
	}

	return nil
}

func scanConversion(row pgx.Row) (*domain.Conversion, error) {
	var c domain.Conversion
	var sourcePlaylistName, targetPlaylistID, targetPlaylistURL, targetPlaylistName, errorMessage *string
	var completedAt *time.Time

	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.SourcePlatform,
		&c.TargetPlatform,
		&c.SourcePlaylistID,
		&sourcePlaylistName,
		&targetPlaylistID,
		&targetPlaylistURL,
		&targetPlaylistName,
		&c.Status,
		&c.TotalTracks,
		&c.ProcessedTracks,
		&c.MatchedTracks,
		&c.FailedTracks,
		&errorMessage,
		&c.CreatedAt,
		&c.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan conversion: %w", err)
	}

	c.SourcePlaylistName = derefString(sourcePlaylistName)
	c.TargetPlaylistID = derefString(targetPlaylistID)
	c.TargetPlaylistURL = derefString(targetPlaylistURL)
	c.TargetPlaylistName = derefString(targetPlaylistName)
	c.ErrorMessage = derefString(errorMessage)
	c.CompletedAt = completedAt

	return &c, nil
}

func scanConversionFromRows(rows pgx.Rows) (*domain.Conversion, error) {
	var c domain.Conversion
	var sourcePlaylistName, targetPlaylistID, targetPlaylistURL, targetPlaylistName, errorMessage *string
	var completedAt *time.Time

	err := rows.Scan(
		&c.ID,
		&c.UserID,
		&c.SourcePlatform,
		&c.TargetPlatform,
		&c.SourcePlaylistID,
		&sourcePlaylistName,
		&targetPlaylistID,
		&targetPlaylistURL,
		&targetPlaylistName,
		&c.Status,
		&c.TotalTracks,
		&c.ProcessedTracks,
		&c.MatchedTracks,
		&c.FailedTracks,
		&errorMessage,
		&c.CreatedAt,
		&c.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan conversion: %w", err)
	}

	c.SourcePlaylistName = derefString(sourcePlaylistName)
	c.TargetPlaylistID = derefString(targetPlaylistID)
	c.TargetPlaylistURL = derefString(targetPlaylistURL)
	c.TargetPlaylistName = derefString(targetPlaylistName)
	c.ErrorMessage = derefString(errorMessage)
	c.CompletedAt = completedAt

	return &c, nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
