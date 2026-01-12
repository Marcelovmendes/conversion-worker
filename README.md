# Conversion Worker

Background worker responsible for asynchronous playlist conversion between streaming platforms (Spotify ↔ YouTube). Part of the [PlaySwap](https://github.com/marcelovmendes/playswap) ecosystem.

## Overview

This worker consumes conversion jobs from a Redis queue, processes them by fetching tracks from the source platform, matching them on the target platform using multiple strategies, and creating the converted playlist.

## How It Works

### Conversion Flow

```
1. PENDING    → Job received from queue
2. FETCHING   → Retrieving tracks from source playlist (Spotify)
3. MATCHING   → Finding equivalent tracks on target platform (YouTube)
4. CREATING   → Creating playlist and adding matched tracks
5. COMPLETED  → Conversion finished successfully
   or FAILED  → Error occurred during any step
```

### Track Matching Strategies

The matcher uses multiple strategies to find the best match for each track:

1. **ISRC Search** (High Confidence) - Uses the International Standard Recording Code when available
2. **Music Search** (Variable Confidence) - Searches by track name and artist
   - **High**: Both artist and title match exactly
   - **Medium**: Either artist or title matches
   - **Low**: General search result

The matcher excludes results containing terms like "cover", "karaoke", "remix", "tutorial" to avoid incorrect matches.

### Real-time Status Updates

Conversion progress is stored in Redis, allowing clients to poll for real-time status updates including:
- Current processing step
- Total/processed/matched/failed track counts
- Progress percentage
- Error messages (if any)

## Tech Stack

- **Go 1.24**
- **PostgreSQL** - Persistent storage for conversions and detailed logs
- **Redis** - Job queue, real-time status, and session management

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Conversion Worker                        │
├─────────────────────────────────────────────────────────────┤
│  cmd/worker/           │ Application entrypoint             │
├─────────────────────────────────────────────────────────────┤
│  internal/                                                   │
│  ├── application/      │ Use cases                          │
│  │   ├── worker        │ Job polling and lifecycle          │
│  │   ├── converter     │ Orchestrates conversion flow       │
│  │   └── matcher       │ Track matching algorithms          │
│  ├── config/           │ Environment-based configuration    │
│  ├── domain/           │ Entities and value objects         │
│  │   ├── conversion    │ Conversion aggregate               │
│  │   ├── track         │ Track entity and matching          │
│  │   ├── playlist      │ Playlist value object              │
│  │   └── platform      │ Platform enum (Spotify/YouTube)    │
│  └── infrastructure/   │ External implementations           │
│      ├── http/         │ Spotify and YouTube API clients    │
│      ├── postgres/     │ Conversion and log repositories    │
│      └── redis/        │ Queue, status, and session stores  │
└─────────────────────────────────────────────────────────────┘
```

## Running

```bash
go run cmd/worker/main.go
```

## Environment Variables

### Redis

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | localhost | Redis server host |
| `REDIS_PORT` | 6379 | Redis server port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_DB` | 0 | Redis database number |

### PostgreSQL

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_HOST` | localhost | PostgreSQL server host |
| `POSTGRES_PORT` | 5432 | PostgreSQL server port |
| `POSTGRES_DATABASE` | playswap | Database name |
| `POSTGRES_USER` | - | Database user |
| `POSTGRES_PASSWORD` | - | Database password |
| `POSTGRES_SSLMODE` | disable | SSL mode |

### External Services

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOTIFY_SERVICE_URL` | http://localhost:8080 | Spotify service base URL |
| `SPOTIFY_SERVICE_TIMEOUT` | 30s | Request timeout |
| `YOUTUBE_SERVICE_URL` | http://localhost:8081 | YouTube service base URL |
| `YOUTUBE_SERVICE_TIMEOUT` | 30s | Request timeout |

### Worker

| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_CONCURRENCY` | 5 | Number of tracks processed in parallel |
| `WORKER_POLL_INTERVAL` | 1s | Queue polling interval |
| `WORKER_JOB_TIMEOUT` | 5m | Maximum time per conversion job |

## Testing

```bash
go test ./...
```

## Related Services

This worker is part of a larger microservices architecture. See the main [PlaySwap repository](https://github.com/marcelovmendes/playswap) for the complete system documentation.
