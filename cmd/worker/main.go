package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/marcelovmendes/playswap/conversion-worker/internal/application"
	appconfig "github.com/marcelovmendes/playswap/conversion-worker/internal/config"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/dynamodb"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/http"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/redis"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/infrastructure/sqs"
	"github.com/marcelovmendes/playswap/conversion-worker/internal/metrics"
)

func main() {
	log.Println("starting conversion worker...")

	cfg := appconfig.Load()

	metrics.StartServer(":9092")
	log.Println("metrics server started on :9092")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient := redis.NewClient(cfg.Redis)
	defer func(redisClient redis.Client) {
		err := redisClient.Close()
		if err != nil {
			log.Printf("error closing redis: %v", err)
		}
	}(redisClient)

	if err := redisClient.Ping(ctx); err != nil {
		log.Fatal("failed to connect to redis: ", err)
	}
	log.Println("connected to redis")

	awsCfg, err := loadAWSConfig(ctx, cfg.AWS)
	if err != nil {
		log.Fatal("failed to load AWS config: ", err)
	}
	log.Println("loaded AWS config")

	queue := sqs.NewJobQueue(awsCfg, cfg.AWS.SQSQueueURL)
	statusStore := redis.NewStatusStore(redisClient)
	sessionStore := redis.NewSessionStore(redisClient)

	conversionRepo := dynamodb.NewConversionRepository(awsCfg, cfg.AWS.DynamoDBConversionsTable)
	logRepo := dynamodb.NewConversionLogRepository(awsCfg, cfg.AWS.DynamoDBLogsTable)

	spotifyClient := http.NewSpotifyClient(cfg.Services.Spotify, sessionStore)
	youtubeClient := http.NewYouTubeClient(cfg.Services.YouTube, sessionStore)

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

func loadAWSConfig(ctx context.Context, awsCfg appconfig.AWSConfig) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(awsCfg.Region),
	}

	if awsCfg.Endpoint != "" {
		opts = append(opts,
			config.WithEndpointResolverWithOptions(
				aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						PartitionID:  "aws",
						URL:          awsCfg.Endpoint,
						SigningRegion: region,
						SigningMethod: "v4",
					}, nil
				}),
			),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		)
	}

	return config.LoadDefaultConfig(ctx, opts...)
}
