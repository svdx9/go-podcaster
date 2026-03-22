-- name: InsertEpisode :one
INSERT INTO episodes (
  uuid, title, description, author, pub_date, file_path, file_name, file_size, mime_type, duration_secs
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetEpisodeByUUID :one
SELECT * FROM episodes
WHERE uuid = ?
LIMIT 1;

-- name: ListEpisodes :many
SELECT * FROM episodes
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListValidEpisodes :many
SELECT * FROM episodes
WHERE duration_secs > 0
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: DeleteEpisode :exec
DELETE FROM episodes
WHERE uuid = ?;

-- name: ListAllEpisodes :many
SELECT * FROM episodes
ORDER BY pub_date DESC;

-- name: ListAllValidEpisodes :many
SELECT * FROM episodes
WHERE duration_secs > 0
ORDER BY pub_date DESC;

-- name: UpdateEpisodeDuration :exec
UPDATE episodes SET duration_secs = ? WHERE uuid = ?;

-- name: ListEpisodesPendingDuration :many
SELECT * FROM episodes WHERE duration_secs = 0;
