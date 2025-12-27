package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/application"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/http"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/postgres"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/redis"
)

func main() {
	log.Println("starting conversion worker...")

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient := redis.NewClient(cfg.Redis)
	defer redisClient.Close()

	if err := redisClient.Ping(ctx); err != nil {
		log.Fatal("failed to connect to redis: ", err)
	}
	log.Println("connected to redis")

	pgClient, err := postgres.NewClient(ctx, cfg.Postgres)
	if err != nil {
		log.Fatal("failed to connect to postgres: ", err)
	}
	defer pgClient.Close()
	log.Println("connected to postgres")

	if err := postgres.RunMigrations(ctx, pgClient); err != nil {
		log.Fatal("failed to run migrations: ", err)
	}
	log.Println("migrations completed")

	queue := redis.NewJobQueue(redisClient)
	statusStore := redis.NewStatusStore(redisClient)

	conversionRepo := postgres.NewConversionRepository(pgClient)
	logRepo := postgres.NewConversionLogRepository(pgClient)

	spotifyClient := http.NewSpotifyClient(cfg.Services.Spotify)
	youtubeClient := http.NewYouTubeClient(cfg.Services.YouTube)

	matcher := application.NewMatcher(youtubeClient)
	converter := application.NewConverter(
		spotifyClient,
		youtubeClient,
		matcher,
		conversionRepo,
		logRepo,
		statusStore,
		cfg.Worker,
	)

	worker := application.NewWorker(queue, converter, cfg.Worker)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("received shutdown signal")
		cancel()
	}()

	worker.Run(ctx)

	log.Println("worker stopped")
}
