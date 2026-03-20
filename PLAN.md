# Go Podcast RSS Backend — Implementation Plan

## Overview
Build a self-contained Go HTTP server to manage podcast episodes: audio upload, delete, list,
SQLite metadata persistence, audio file serving, and an iTunes-compatible `/feed.xml`.

---

## Step 1: Scaffold & Module Init

- [x] Run `go mod init` in `/`
- [x] Create `Makefile` with all targets: `tools-dir`, `tools-install`, `generate`,
      `migrate-up`, `migrate-down`, `build`, `test`, `lint`, `fmt`, `dev`
- [x] Create `.air.toml` for hot reload
- [x] Create `.gitignore` (ignore `tools/bin/`, `*.db`, `uploads/`)

---

## Step 2: OpenAPI Spec + oapi-codegen Config

**Files:**
- [x] `docs/schema/v1/podcast.yaml` — OpenAPI 3.1 spec with all 5 endpoints:
  - `POST /v1/episodes`
  - `GET /v1/episodes`
  - `DELETE /v1/episodes/{uuid}`
  - `GET /feed.xml`
  - `GET /files/{uuid}/{filename}`
- [x] `docs/schema/v1/config.yaml` — oapi-codegen generation config
  - `package: api`
  - `generate: chi-server: true, models: true, embedded-spec: true`
  - `output: internal/api/v1/api.gen.go`

**Notes:**
- Error envelope: `{ "code": "string", "message": "string" }`
- `POST /v1/episodes` uses `multipart/form-data` (fields: `title`, `description`, `author`, `pub_date`, `file`)
- `GET /v1/episodes` supports `limit` and `offset` query params
- All 4xx/5xx responses reference the error schema

---

## Step 3: Database Migrations

**Files:**
- [x] `sql/migrations/000001_create_episodes.up.sql`
- [x] `sql/migrations/000001_create_episodes.down.sql`

**Schema:**
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
  duration_secs INTEGER,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## Step 4: sqlc Queries + Config

**Files:**
- [x] `sqlc.yaml` — dialect: sqlite, engine: modernc, output to `internal/db/queries/`
- [x] `sql/queries/episodes.sql` — named queries:
  - `InsertEpisode`
  - `GetEpisodeByUUID`
  - `ListEpisodes` (with `LIMIT`/`OFFSET`)
  - `DeleteEpisode`
  - `ListAllEpisodes` (for feed; ordered by `pub_date DESC`)

---

## Step 5: Config Package

**File:** `internal/config/config.go`

- Struct with all fields typed (no raw strings outside this package)
- `Load() (Config, error)` — reads all env vars, validates required ones, applies defaults
- Fails fast with descriptive error if `BASE_URL`, `PODCAST_TITLE`, `PODCAST_DESCRIPTION`,
  or `PODCAST_AUTHOR` are missing
- `Redacted() string` method for safe diagnostic logging

| Env Var               | Required | Default        |
| --------------------- | -------- | -------------- |
| `PORT`                | No       | `8080`         |
| `DB_PATH`             | No       | `./podcast.db` |
| `UPLOAD_DIR`          | No       | `./uploads`    |
| `BASE_URL`            | Yes      | —              |
| `PODCAST_TITLE`       | Yes      | —              |
| `PODCAST_DESCRIPTION` | Yes      | —              |
| `PODCAST_AUTHOR`      | Yes      | —              |
| `PODCAST_LANGUAGE`    | No       | `en-us`        |
| `PODCAST_CATEGORY`    | No       | `Technology`   |
| `PODCAST_IMAGE_URL`   | No       | —              |
| `LOG_LEVEL`           | No       | `info`         |

---

## Step 6: DB Package (SQLite Open + sqlc wiring)

**Files:**
- `internal/db/db.go`
  - `Open(ctx, dbPath) (*sql.DB, error)` — opens SQLite via `modernc.org/sqlite`, pings
  - Enables WAL mode and foreign keys via PRAGMA
- `internal/db/queries/` — sqlc-generated (do not edit)

---

## Step 7: Audio Package

**Files:**
- `internal/audio/validate.go`
  - `AllowedMIMETypes` sentinel set
  - `DetectMIME(data []byte, filename string) (string, error)` — magic-byte detection via
    `net/http.DetectContentType` cross-checked against file extension
  - Returns error if MIME not in allowlist

- `internal/audio/meta.go`
  - `Meta` struct: `Title`, `Artist`, `DurationSecs`
  - `Extract(r io.ReadSeeker) (Meta, error)` — uses `github.com/dhowden/tag`
  - Gracefully handles files with no tags (returns zero-value Meta, no error)

**Allowed MIME types:**
`audio/mpeg`, `audio/x-m4a`, `audio/mp4`, `audio/aac`,
`audio/x-aiff`, `audio/aiff`, `audio/wav`, `audio/x-wav`

---

## Step 8: Episode Repository Interface + DB Implementation

**Files:**
- `internal/episode/repository/repository.go`
  - `Repository` interface (owned by feature):
    ```go
    type Repository interface {
        Insert(ctx, Episode) error
        GetByUUID(ctx, uuid string) (Episode, error)
        List(ctx, limit, offset int) ([]Episode, error)
        Delete(ctx, uuid string) error
        ListAll(ctx) ([]Episode, error)
    }
    ```
  - `Episode` domain struct (mirrors DB columns; no DB types in signature)
  - `var ErrNotFound = errors.New("not found")`

- `internal/db/episode_repo.go` — implements `repository.Repository` using sqlc queries
  - Maps sqlc-generated types ↔ domain `Episode`
  - Wraps `sql.ErrNoRows` as `repository.ErrNotFound`

---

## Step 9: Episode Service

**File:** `internal/episode/service/service.go`

- `Service` struct with `Repository`, `UploadDir` injected
- Methods (all accept `context.Context`):
  - `Upload(ctx, UploadRequest) (Episode, error)`
    1. Validate: `description` and `file` required
    2. Detect MIME from bytes; reject with `ErrUnsupportedMedia` if not allowed
    3. Extract audio tags
    4. Merge: caller-supplied fields win; `title` still empty → `ErrMissingTitle`
    5. Generate UUID; `mkdir {UPLOAD_DIR}/{uuid}/`; write file
    6. Insert DB row in transaction
    7. On any failure after file write: delete the directory
  - `Delete(ctx, uuid string) error`
    1. Get episode (→ 404 if not found)
    2. Delete DB row
    3. Remove `{UPLOAD_DIR}/{uuid}/` from disk
  - `List(ctx, limit, offset int) ([]Episode, error)`

- Sentinel errors: `ErrMissingTitle`, `ErrUnsupportedMedia`

---

## Step 10: Episode HTTP Handlers + DTO Mapping

**File:** `internal/episode/api/handler.go`

- Implements `oapi-codegen`-generated `StrictServerInterface`
- `Handler` struct with injected `*service.Service`
- Per-endpoint:
  - `PostV1Episodes` — parse multipart, call service, return 201
  - `GetV1Episodes` — parse query params, call service, return 200
  - `DeleteV1EpisodesUuid` — call service, return 204
- Error mapping:
  - `service.ErrNotFound` → 404
  - `service.ErrMissingTitle` → 400
  - `service.ErrUnsupportedMedia` → 415
  - All others → 500
- DTO helpers: `episodeToResponse(Episode) api.Episode`

---

## Step 11: Feed Package

**File:** `internal/feed/feed.go`

- iTunes RSS 2.0 structs with correct `encoding/xml` tags:
  - `Channel`, `Item`, `Enclosure`, `ItunesImage`
  - Namespace: `xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"`
- `Generator` struct with `Repository`, `Config` injected
- `Render(ctx, w io.Writer) error`
  1. `ListAll` from repo
  2. Build channel + items
  3. Enclosure URL: `{BASE_URL}/files/{uuid}/{filename}`
  4. `<itunes:duration>` only when `duration_secs` non-null (format: `HH:MM:SS`)
  5. `<itunes:author>` per-item only when `author` non-empty
  6. Write XML with `xml.NewEncoder`

---

## Step 12: HTTP Server Wiring

**File:** `internal/http/server.go`

- `New(cfg, episodeHandler, feedGen) *Server`
- chi router with:
  - `slog` request logger middleware
  - `recoverer` middleware
  - Routes mounted: `/v1/...` via oapi-codegen strict handler, `/feed.xml`, `/files/{uuid}/{filename}`
- `Server.Start(ctx) error` — `http.ListenAndServe` with graceful shutdown on ctx cancel
- File serving: `http.ServeContent` with correct `Content-Type` from stored `mime_type`

---

## Step 13: Composition Root

**File:** `cmd/server/main.go`

- `main()`:
  1. `config.Load()`
  2. Setup `slog` with configured level
  3. `db.Open(ctx, cfg.DBPath)`
  4. Construct `db.EpisodeRepository`
  5. Construct `episode/service.Service`
  6. Construct `episode/api.Handler`
  7. Construct `feed.Generator`
  8. Construct `http/server.New(...)`
  9. `server.Start(ctx)` with `signal.NotifyContext` for SIGINT/SIGTERM

---

## Step 14: go.mod Dependencies

Run `go get` for:
- `github.com/go-chi/chi/v5`
- `modernc.org/sqlite`
- `github.com/dhowden/tag`
- `github.com/google/uuid`
- `github.com/oapi-codegen/runtime`

Dev/tools (installed via `make tools-install`, NOT in go.mod):
- `github.com/air-verse/air`
- `mvdan.cc/gofumpt`
- `github.com/golangci/golangci-lint/cmd/golangci-lint`
- `github.com/golang-migrate/migrate/v4/cmd/migrate`
- `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`
- `github.com/sqlc-dev/sqlc/cmd/sqlc`

---

## Step 15: Tests

**Files to create:**
- `internal/audio/validate_test.go` — table-driven: allowed/rejected MIME types
- `internal/audio/meta_test.go` — extract from fixture file with/without tags
- `internal/feed/feed_test.go` — render with known episodes, assert XML shape
- `internal/episode/service/service_test.go` — mock repository; test upload validation rules

---

## Verification Checklist

- [ ] `make tools-install && make generate && make build` succeeds cleanly
- [ ] `make migrate-up` against fresh DB — no errors
- [ ] `make test` — all tests pass
- [ ] Upload MP3 with no ID3 title + no `title` field → 400
- [ ] Upload MP3 with `description` and `title` → 201 with UUID
- [ ] `GET /feed.xml` → valid iTunes RSS
- [ ] `GET /files/{uuid}/{filename}` → streams audio with correct `Content-Type`
- [ ] `DELETE /v1/episodes/{uuid}` → 204; file deleted; absent from list
- [ ] Upload PDF → 415
- [ ] `make lint` — zero errors

---

## Dependency Direction (reminder)

```
http/server.go
  → episode/api (handlers)
    → episode/service (use-cases)
      → episode/repository (interface + domain types)
        ← internal/db (implements interface)
  → feed/
  → config/
```

`episode/` MUST NOT import `internal/db` directly.

# TODO

Change the openapi spec:

OpenAPI 3.1 spec from this:
  - `GET /files/{uuid}/{filename}`
to this:
  - `GET /files/{uuid}`

