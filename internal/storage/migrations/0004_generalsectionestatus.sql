CREATE TABLE general_section_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    section_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL,
    content_hash TEXT NOT NULL,
    completed_at DATETIME,
    error TEXT NOT NULL,
    started_at DATETIME,
    UNIQUE (section_id)
);
