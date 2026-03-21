package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/application"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

const logTTLDays = 30

type logItem struct {
	ID                string               `dynamodbav:"id"`
	ConversionID      string               `dynamodbav:"conversionId"`
	Step              domain.ConversionStep `dynamodbav:"step"`
	Status            domain.LogStatus      `dynamodbav:"status"`
	SourceTrackID     string               `dynamodbav:"sourceTrackId,omitempty"`
	SourceTrackName   string               `dynamodbav:"sourceTrackName,omitempty"`
	SourceTrackArtist string               `dynamodbav:"sourceTrackArtist,omitempty"`
	TargetTrackID     string               `dynamodbav:"targetTrackId,omitempty"`
	TargetTrackName   string               `dynamodbav:"targetTrackName,omitempty"`
	ErrorMessage      string               `dynamodbav:"errorMessage,omitempty"`
	CreatedAt         string               `dynamodbav:"createdAt"`
	TTL               int64                `dynamodbav:"ttl"`
}

type conversionLogRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewConversionLogRepository(cfg aws.Config, tableName string) application.ConversionLogRepository {
	return &conversionLogRepository{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}
}

func (r *conversionLogRepository) Create(ctx context.Context, l *domain.ConversionLog) error {
	item := toLogItem(l)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	return nil
}

func (r *conversionLogRepository) CreateBatch(ctx context.Context, logs []*domain.ConversionLog) error {
	if len(logs) == 0 {
		return nil
	}

	for i := 0; i < len(logs); i += 25 {
		end := i + 25
		if end > len(logs) {
			end = len(logs)
		}

		var requests []types.WriteRequest
		for _, l := range logs[i:end] {
			item := toLogItem(l)
			av, err := attributevalue.MarshalMap(item)
			if err != nil {
				return fmt.Errorf("failed to marshal log: %w", err)
			}
			requests = append(requests, types.WriteRequest{
				PutRequest: &types.PutRequest{Item: av},
			})
		}

		_, err := r.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				r.tableName: requests,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to batch write logs: %w", err)
		}
	}

	return nil
}

func toLogItem(l *domain.ConversionLog) logItem {
	return logItem{
		ID:                l.ID,
		ConversionID:      l.ConversionID,
		Step:              l.Step,
		Status:            l.Status,
		SourceTrackID:     l.SourceTrackID,
		SourceTrackName:   l.SourceTrackName,
		SourceTrackArtist: l.SourceTrackArtist,
		TargetTrackID:     l.TargetTrackID,
		TargetTrackName:   l.TargetTrackName,
		ErrorMessage:      l.ErrorMessage,
		CreatedAt:         l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		TTL:               time.Now().Add(logTTLDays * 24 * time.Hour).Unix(),
	}
}
