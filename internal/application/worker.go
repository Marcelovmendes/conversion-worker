package application

import (
	"context"
	"log"
	"time"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/redis"
)

type Worker interface {
	Run(ctx context.Context)
}

type worker struct {
	queue     redis.JobQueue
	converter Converter
	config    config.WorkerConfig
}

func NewWorker(
	queue redis.JobQueue,
	converter Converter,
	cfg config.WorkerConfig,
) Worker {
	return &worker{
		queue:     queue,
		converter: converter,
		config:    cfg,
	}
}

func (w *worker) Run(ctx context.Context) {
	log.Printf("worker started, polling every %v", w.config.PollInterval)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down...")
			return
		case <-ticker.C:
			w.processNextJob(ctx)
		}
	}
}

func (w *worker) processNextJob(ctx context.Context) {
	job, err := w.queue.Pop(ctx, w.config.PollInterval)
	if err != nil {
		log.Printf("error polling queue: %v", err)
		return
	}

	if job == nil {
		return
	}

	log.Printf("received job %s: %s -> %s", job.JobID, job.SourcePlatform, job.TargetPlatform)

	jobCtx, cancel := context.WithTimeout(ctx, w.config.JobTimeout)
	defer cancel()

	if err := w.converter.Convert(jobCtx, job); err != nil {
		log.Printf("job %s failed: %v", job.JobID, err)
	}
}
