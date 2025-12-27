package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type ConversionLogRepository interface {
	Create(ctx context.Context, log *domain.ConversionLog) error
	CreateBatch(ctx context.Context, logs []*domain.ConversionLog) error
	FindByConversionID(ctx context.Context, conversionID string) ([]*domain.ConversionLog, error)
	FindFailedByConversionID(ctx context.Context, conversionID string) ([]*domain.ConversionLog, error)
}

type conversionLogRepository struct {
	pool *pgxpool.Pool
}

func NewConversionLogRepository(client Client) ConversionLogRepository {
	return &conversionLogRepository{pool: client.GetPool()}
}

func (r *conversionLogRepository) Create(ctx context.Context, log *domain.ConversionLog) error {
	query := `
		INSERT INTO conversion_logs (
			id, conversion_id, step, status,
			source_track_id, source_track_name, source_track_artist,
			target_track_id, target_track_name,
			error_message, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.ConversionID,
		log.Step,
		log.Status,
		nullableString(log.SourceTrackID),
		nullableString(log.SourceTrackName),
		nullableString(log.SourceTrackArtist),
		nullableString(log.TargetTrackID),
		nullableString(log.TargetTrackName),
		nullableString(log.ErrorMessage),
		log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create conversion log: %w", err)
	}

	return nil
}

func (r *conversionLogRepository) CreateBatch(ctx context.Context, logs []*domain.ConversionLog) error {
	if len(logs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO conversion_logs (
			id, conversion_id, step, status,
			source_track_id, source_track_name, source_track_artist,
			target_track_id, target_track_name,
			error_message, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	for _, log := range logs {
		batch.Queue(query,
			log.ID,
			log.ConversionID,
			log.Step,
			log.Status,
			nullableString(log.SourceTrackID),
			nullableString(log.SourceTrackName),
			nullableString(log.SourceTrackArtist),
			nullableString(log.TargetTrackID),
			nullableString(log.TargetTrackName),
			nullableString(log.ErrorMessage),
			log.CreatedAt,
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range logs {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to execute batch insert: %w", err)
		}
	}

	return nil
}

func (r *conversionLogRepository) FindByConversionID(ctx context.Context, conversionID string) ([]*domain.ConversionLog, error) {
	query := `
		SELECT
			id, conversion_id, step, status,
			source_track_id, source_track_name, source_track_artist,
			target_track_id, target_track_name,
			error_message, created_at
		FROM conversion_logs
		WHERE conversion_id = $1
		ORDER BY created_at ASC
	`

	return r.queryLogs(ctx, query, conversionID)
}

func (r *conversionLogRepository) FindFailedByConversionID(ctx context.Context, conversionID string) ([]*domain.ConversionLog, error) {
	query := `
		SELECT
			id, conversion_id, step, status,
			source_track_id, source_track_name, source_track_artist,
			target_track_id, target_track_name,
			error_message, created_at
		FROM conversion_logs
		WHERE conversion_id = $1 AND status = 'FAILED'
		ORDER BY created_at ASC
	`

	return r.queryLogs(ctx, query, conversionID)
}

func (r *conversionLogRepository) queryLogs(ctx context.Context, query string, args ...any) ([]*domain.ConversionLog, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.ConversionLog
	for rows.Next() {
		log, err := scanLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

func scanLog(rows pgx.Rows) (*domain.ConversionLog, error) {
	var log domain.ConversionLog
	var sourceTrackID, sourceTrackName, sourceTrackArtist *string
	var targetTrackID, targetTrackName, errorMessage *string

	err := rows.Scan(
		&log.ID,
		&log.ConversionID,
		&log.Step,
		&log.Status,
		&sourceTrackID,
		&sourceTrackName,
		&sourceTrackArtist,
		&targetTrackID,
		&targetTrackName,
		&errorMessage,
		&log.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan log: %w", err)
	}

	log.SourceTrackID = derefString(sourceTrackID)
	log.SourceTrackName = derefString(sourceTrackName)
	log.SourceTrackArtist = derefString(sourceTrackArtist)
	log.TargetTrackID = derefString(targetTrackID)
	log.TargetTrackName = derefString(targetTrackName)
	log.ErrorMessage = derefString(errorMessage)

	return &log, nil
}
