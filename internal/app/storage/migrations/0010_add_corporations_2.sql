CREATE TABLE corporations (
    id INTEGER PRIMARY KEY NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id) REFERENCES eve_corporations (id) ON DELETE CASCADE
);

CREATE TABLE corporation_section_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    section_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL,
    content_hash TEXT NOT NULL,
    completed_at DATETIME,
    error TEXT NOT NULL,
    started_at DATETIME,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, section_id)
);

CREATE INDEX corporation_section_status_idx1 ON corporation_section_status (corporation_id);

CREATE INDEX corporation_section_status_idx2 ON corporation_section_status (section_id);