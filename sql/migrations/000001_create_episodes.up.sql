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
