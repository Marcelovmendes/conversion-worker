package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/application"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type jobQueue struct {
	client   *sqs.Client
	queueURL string
}

func NewJobQueue(cfg aws.Config, queueURL string) application.JobQueue {
	return &jobQueue{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
	}
}

func (q *jobQueue) Receive(ctx context.Context) (*application.QueuedJob, error) {
	out, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &q.queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to receive message: %w", err)
	}

	if len(out.Messages) == 0 {
		return nil, nil
	}

	msg := out.Messages[0]

	var job domain.ConversionJob
	if err := json.Unmarshal([]byte(*msg.Body), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &application.QueuedJob{
		Job:           &job,
		ReceiptHandle: *msg.ReceiptHandle,
	}, nil
}

func (q *jobQueue) Delete(ctx context.Context, receiptHandle string) error {
	_, err := q.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &q.queueURL,
		ReceiptHandle: &receiptHandle,
	})
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}
