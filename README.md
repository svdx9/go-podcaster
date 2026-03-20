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
make migrate-up DB_PATH=./podcast.db
make build
./bin/server
```

Or with hot reload:

```bash
make dev
```
