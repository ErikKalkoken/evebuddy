CREATE TABLE corporation_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, character_id)
);

CREATE INDEX corporation_members_idx1 ON corporation_members (corporation_id);
CREATE INDEX corporation_members_idx2 ON corporation_members (character_id);
