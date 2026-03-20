# Go Podcaster

A podcast RSS backend server for managing podcast episodes with audio upload, SQLite metadata persistence, and iTunes-compatible RSS feed generation.

## API Endpoints

### Upload a new episode

```bash
curl -X POST http://localhost:9871/v1/episodes \
  -F "title=My First Episode" \
  -F "description=This is my first podcast episode" \
  -F "author=John Doe" \
  -F "pub_date=2026-03-18T10:00:00Z" \
  -F "file=@episode.mp3"
```

### List episodes

```bash
curl http://localhost:9871/v1/episodes
```

With pagination:

```bash
curl "http://localhost:9871/v1/episodes?limit=5&offset=0"
```

### Delete an episode

```bash
curl -X DELETE http://localhost:9871/v1/episodes/<uuid>
```

### Get RSS feed

```bash
curl http://localhost:9871/feed.xml
```

### Download an audio file

```bash
curl http://localhost:9871/files/<uuid> -o episode.mp3
```

## Running

```bash
make build
./bin/server
```

The database schema is created automatically on startup if the database is not initialized.

Or with hot reload:

```bash
make dev
```

## Implementation Details

### Architecture

The server follows a layered architecture with clear separation of concerns:

```
cmd/server/main.go
    ├── config (env-based configuration)
    ├── http (chi router, middleware)
    ├── episode/api (HTTP handlers)
    ├── episode/service (business logic)
    ├── episode/repository (data access)
    ├── feed (RSS generation)
    ├── file (content-addressable storage)
    └── db (SQLite + sqlc)
```

### Technology Stack

- **HTTP Server**: [chi](https://go-chi.io/) - lightweight router with middleware support
- **Database**: SQLite with [sqlc](https://sqlc.dev/) for type-safe queries
- **Audio Processing**: [id3v2](https://github.com/mikkyang/id3v2) for duration extraction
- **Logging**: Go's built-in `slog` package
- **API Spec**: [go-enum](https://github.com/abice/go-enum) + OpenAPI v3 code generation

### Components

**HTTP Server** (`internal/http/server.go`)
- Chi-based router with structured logging middleware
- Request timeouts: 30s read, 30s write, 120s idle
- Graceful shutdown with signal handling

**Episode Service** (`internal/episode/service/service.go`)
- Handles episode upload, listing, and deletion
- Validates audio files (MP3, M4A, WAV supported)
- Extracts ID3 metadata for duration

**File Store** (`internal/file/file_store.go`)
- Content-addressable storage using SHA-1 UUID derived from file content
- Atomic writes via temp file + rename
- Same-filesystem guarantee for atomic operations

**RSS Feed Generator** (`internal/feed/feed.go`)
- iTunes-compatible RSS 2.0 with namespace
- Items sorted by pub_date (newest first)
- Proper duration formatting (H:MM:SS or M:SS)

### Database Schema

```sql
CREATE TABLE episodes (
  uuid          TEXT PRIMARY KEY,
  title         TEXT NOT NULL,
  description   TEXT NOT NULL,
  author        TEXT,
  pub_date      DATETIME NOT NULL,
  file_path     TEXT NOT NULL,
  file_name     TEXT NOT NULL,
  file_size     INTEGER NOT NULL,
  mime_type     TEXT NOT NULL,
  duration_secs INTEGER NOT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Configuration

Environment variables (all required unless noted):

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Hostname for URL generation | (required) |
| `BASE_URL` | Public podcast URL | (required) |
| `PORT` | Listen port | `8080` |
| `ENV` | deployment environment | `development` |
| `LOG_LEVEL` | DEBUG, INFO, WARN, ERROR | INFO (prod) / DEBUG (dev) |
| `DB_PATH` | SQLite database path | `./podcast.db` |
| `UPLOAD_DIR` | Audio file storage | `./uploads` |
| `PODCAST_TITLE` | Podcast title | (required) |
| `PODCAST_DESCRIPTION` | Podcast description | (required) |
| `PODCAST_AUTHOR` | Podcast author | (required) |
| `PODCAST_LANGUAGE` | iTunes language code | `en-us` |
| `PODCAST_CATEGORY` | iTunes category | `Technology` |
| `PODCAST_IMAGE_URL` | Cover image URL | (optional) |

### Running Locally

```bash
# Create uploads directory
mkdir -p uploads

# Run with environment
HOST=localhost PORT=9871 \
  BASE_URL=http://localhost:9871 \
  PODCAST_TITLE="My Podcast" \
  PODCAST_DESCRIPTION="A great show" \
  PODCAST_AUTHOR="John Doe" \
  go run ./cmd/server
```
