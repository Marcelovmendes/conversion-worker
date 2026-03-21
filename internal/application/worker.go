package application

import (
	"context"
	"log"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/domain"
)

type QueuedJob struct {
	Job           *domain.ConversionJob
	ReceiptHandle string
}

type JobQueue interface {
	Receive(ctx context.Context) (*QueuedJob, error)
	Delete(ctx context.Context, receiptHandle string) error
}

type Worker interface {
	Run(ctx context.Context)
}

type worker struct {
	queue     JobQueue
	converter Converter
	config    config.WorkerConfig
}

func NewWorker(queue JobQueue, converter Converter, cfg config.WorkerConfig) Worker {
	return &worker{
		queue:     queue,
		converter: converter,
		config:    cfg,
	}
}

func (w *worker) Run(ctx context.Context) {
	log.Println("worker started with long-polling")

	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down...")
			return
		default:
			w.processNextJob(ctx)
		}
	}
}

func (w *worker) processNextJob(ctx context.Context) {
	queued, err := w.queue.Receive(ctx)
	if err != nil {
		log.Printf("error polling queue: %v", err)
		return
	}

	if queued == nil {
		return
	}

	job := queued.Job
	log.Printf("received job %s: %s -> %s", job.JobID, job.SourcePlatform, job.TargetPlatform)

	jobCtx, cancel := context.WithTimeout(ctx, w.config.JobTimeout)
	defer cancel()

	if err := w.converter.Convert(jobCtx, job); err != nil {
		log.Printf("job %s failed: %v", job.JobID, err)
		return
	}

	if err := w.queue.Delete(ctx, queued.ReceiptHandle); err != nil {
		log.Printf("failed to delete job %s from queue: %v", job.JobID, err)
	}
}
