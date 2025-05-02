CREATE TABLE character_roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    UNIQUE (character_id, name)
);

CREATE INDEX character_roles_jobs_idx1 ON character_roles (character_id);
