CREATE TABLE cache(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    key TEXT NOT NULL,
    value BLOB NOT NULL,
    UNIQUE(key)
);

CREATE INDEX cache_idx1 ON cache (expires_at);