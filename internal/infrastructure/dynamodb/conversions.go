package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/application"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type conversionItem struct {
	ID                 string                   `dynamodbav:"id"`
	UserID             string                   `dynamodbav:"userId"`
	SourcePlatform     domain.Platform          `dynamodbav:"sourcePlatform"`
	TargetPlatform     domain.Platform          `dynamodbav:"targetPlatform"`
	SourcePlaylistID   string                   `dynamodbav:"sourcePlaylistId"`
	SourcePlaylistName string                   `dynamodbav:"sourcePlaylistName,omitempty"`
	TargetPlaylistID   string                   `dynamodbav:"targetPlaylistId,omitempty"`
	TargetPlaylistURL  string                   `dynamodbav:"targetPlaylistUrl,omitempty"`
	TargetPlaylistName string                   `dynamodbav:"targetPlaylistName,omitempty"`
	Status             domain.ConversionStatus  `dynamodbav:"status"`
	TotalTracks        int                      `dynamodbav:"totalTracks"`
	ProcessedTracks    int                      `dynamodbav:"processedTracks"`
	MatchedTracks      int                      `dynamodbav:"matchedTracks"`
	FailedTracks       int                      `dynamodbav:"failedTracks"`
	ErrorMessage       string                   `dynamodbav:"errorMessage,omitempty"`
	CreatedAt          string                   `dynamodbav:"createdAt"`
	UpdatedAt          string                   `dynamodbav:"updatedAt"`
	CompletedAt        string                   `dynamodbav:"completedAt,omitempty"`
}

type conversionRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewConversionRepository(cfg aws.Config, tableName string) application.ConversionRepository {
	return &conversionRepository{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}
}

func (r *conversionRepository) Create(ctx context.Context, c *domain.Conversion) error {
	item := toConversionItem(c)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal conversion: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to create conversion: %w", err)
	}
	return nil
}

func (r *conversionRepository) Update(ctx context.Context, c *domain.Conversion) error {
	return r.Create(ctx, c)
}

func toConversionItem(c *domain.Conversion) conversionItem {
	item := conversionItem{
		ID:                 c.ID,
		UserID:             c.UserID,
		SourcePlatform:     c.SourcePlatform,
		TargetPlatform:     c.TargetPlatform,
		SourcePlaylistID:   c.SourcePlaylistID,
		SourcePlaylistName: c.SourcePlaylistName,
		TargetPlaylistID:   c.TargetPlaylistID,
		TargetPlaylistURL:  c.TargetPlaylistURL,
		TargetPlaylistName: c.TargetPlaylistName,
		Status:             c.Status,
		TotalTracks:        c.TotalTracks,
		ProcessedTracks:    c.ProcessedTracks,
		MatchedTracks:      c.MatchedTracks,
		FailedTracks:       c.FailedTracks,
		ErrorMessage:       c.ErrorMessage,
		CreatedAt:          c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if c.CompletedAt != nil {
		item.CompletedAt = c.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return item
}
