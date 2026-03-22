# Plan: Background ffprobe Metadata Extraction

## Context

The current `service.Upload` blocks on ffprobe (spawns subprocess, copies to temp file, parses JSON output) before returning. This is slow and unnecessary — the file is already being saved to disk, so ffprobe can run against the persisted file after the upload returns. The goal: upload saves the file + inserts a DB row with `duration_secs = 0`, then a background goroutine runs ffprobe and updates the row.

## Implementation Steps

### 1. Add SQL queries for update + pending lookup
**File:** `sql/queries/episodes.sql`

```sql
-- name: UpdateEpisodeDuration :exec
UPDATE episodes SET duration_secs = ? WHERE uuid = ?;

-- name: ListEpisodesPendingDuration :many
SELECT * FROM episodes WHERE duration_secs = 0;
```

Then run `sqlc generate` to regenerate `internal/db/queries/`.

### 2. Extend Repository interface + implementation
**File:** `internal/episode/repository/repository.go` — add to interface:
- `UpdateDuration(ctx context.Context, UUID uuid.UUID, durationSecs int) error`

**File:** `internal/db/episode_repo.go` — implement using generated `UpdateEpisodeDuration`.

### 3. Split `audio.Extract` into two public functions
**File:** `internal/audio/meta.go`

- **`ReadTags(r io.ReadSeeker) (Meta, error)`** — fast, in-memory ID3 tag parsing (title, artist). Used during upload.
- **`ProbeDuration(r io.Reader) (int, error)`** — public wrapper around existing `ffprobeDuration`. Used by background worker.
- Keep `Extract` intact (it calls both internally) for backward compat.

### 4. Refactor `service.Upload` — remove ffprobe from hot path
**File:** `internal/episode/service/service.go`

- Replace `audio.Extract(req.File)` with `audio.ReadTags(req.File)` (tags only, no ffprobe)
- Insert episode with `DurationSecs: 0`
- After successful insert, enqueue `probeJob{UUID}` on a buffered channel
- Remove `ErrZeroDurationEpisode` and `ErrFfprobeNotFound` sentinel errors (no longer produced during upload)

### 5. Add background probe worker to Service
**File:** `internal/episode/service/service.go`

```go
type probeJob struct { UUID uuid.UUID }
```

- Add `probeQueue chan probeJob` field to `Service`
- `StartBackgroundProbe(ctx context.Context)` — goroutine reads from channel, calls `file.Store.ReadSeekFile` to re-open file from disk, runs `audio.ProbeDuration`, then `repo.UpdateDuration`
- Non-blocking enqueue with select/default (log warning if queue full)

### 6. Update handler — remove dead error mappings
**File:** `internal/episode/api/handler.go`

Remove `ErrZeroDurationEpisode` and `ErrFfprobeNotFound` cases from `handleServiceError`.

### 7. Guard feed generator against zero duration
**File:** `internal/feed/feed.go`

In `episodeToItem`: only set `Duration` if `DurationSecs > 0`. The `omitempty` XML tag already handles omission.

### 8. Wire up in main.go
**File:** `cmd/server/main.go`

- Call `svc.StartBackgroundProbe(ctx)` after creating the service
- On startup, query `ListEpisodesPendingDuration` and enqueue any unprocessed episodes (crash recovery)

### 9. Update tests
- `internal/episode/service/service_test.go` — add `UpdateDuration` to mock repo, remove references to `ErrZeroDurationEpisode`
- `internal/audio/meta_test.go` — add tests for `ReadTags` and `ProbeDuration`

## File Change Summary

| File | Change |
|------|--------|
| `sql/queries/episodes.sql` | Add UPDATE + pending-list queries |
| `internal/db/queries/*` | Regenerate via `sqlc generate` |
| `internal/episode/repository/repository.go` | Add `UpdateDuration` to interface |
| `internal/db/episode_repo.go` | Implement `UpdateDuration` |
| `internal/audio/meta.go` | Export `ReadTags` + `ProbeDuration` |
| `internal/episode/service/service.go` | Refactor Upload, add background worker |
| `internal/episode/api/handler.go` | Remove dead error mappings |
| `internal/feed/feed.go` | Guard zero duration |
| `cmd/server/main.go` | Wire background worker + crash recovery |
| `internal/episode/service/service_test.go` | Update mocks + tests |
| `internal/audio/meta_test.go` | Test new public functions |

## Design Decisions

- **Tags stay synchronous**: `readTags` is fast (in-memory parsing) and provides title/artist fallback needed during upload. Only ffprobe moves to background.
- **Simple goroutine + channel**: No external job queue needed — a single goroutine with a buffered channel is sufficient for this workload.
- **No schema change**: Keep `duration_secs INTEGER NOT NULL`. Upload explicitly inserts 0. No migration or ALTER TABLE needed.
- **0 = pending**: `duration_secs = 0` means "not yet processed". Existing rows already have real durations so no conflict.
- **Crash recovery**: On startup, re-enqueue episodes with `duration_secs = 0` so no uploads are permanently stuck.

## Verification

1. `sqlc generate` succeeds
2. `go build ./...` compiles
3. `go test ./...` passes
4. Manual test: upload an MP3 — response returns immediately with `duration_secs: 0`, then GET the episode shortly after and see the real duration populated
5. Feed XML omits `<itunes:duration>` for pending episodes, shows it for completed ones
