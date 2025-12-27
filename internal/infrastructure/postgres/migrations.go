package postgres

import (
	"context"
	"fmt"
)

const createConversionsTableSQL = `
CREATE TABLE IF NOT EXISTS conversions (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    source_platform VARCHAR(50) NOT NULL,
    target_platform VARCHAR(50) NOT NULL,
    source_playlist_id VARCHAR(255) NOT NULL,
    source_playlist_name VARCHAR(500),
    target_playlist_id VARCHAR(255),
    target_playlist_url VARCHAR(500),
    target_playlist_name VARCHAR(500),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    total_tracks INT NOT NULL DEFAULT 0,
    processed_tracks INT NOT NULL DEFAULT 0,
    matched_tracks INT NOT NULL DEFAULT 0,
    failed_tracks INT NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_conversions_user_id ON conversions(user_id);
CREATE INDEX IF NOT EXISTS idx_conversions_status ON conversions(status);
CREATE INDEX IF NOT EXISTS idx_conversions_created_at ON conversions(created_at DESC);
`

const createConversionLogsTableSQL = `
CREATE TABLE IF NOT EXISTS conversion_logs (
    id UUID PRIMARY KEY,
    conversion_id UUID NOT NULL REFERENCES conversions(id) ON DELETE CASCADE,
    step VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    source_track_id VARCHAR(255),
    source_track_name VARCHAR(500),
    source_track_artist VARCHAR(500),
    target_track_id VARCHAR(255),
    target_track_name VARCHAR(500),
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_conversion_logs_conversion_id ON conversion_logs(conversion_id);
`

func RunMigrations(ctx context.Context, client Client) error {
	pool := client.GetPool()

	if _, err := pool.Exec(ctx, createConversionsTableSQL); err != nil {
		return fmt.Errorf("failed to create conversions table: %w", err)
	}

	if _, err := pool.Exec(ctx, createConversionLogsTableSQL); err != nil {
		return fmt.Errorf("failed to create conversion_logs table: %w", err)
	}

	return nil
}
