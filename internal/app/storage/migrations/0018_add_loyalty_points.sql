CREATE TABLE character_loyalty_point_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    corporation_id INTEGER NOT NULL,
    loyalty_points INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    FOREIGN KEY (corporation_id) REFERENCES eve_corporations (id) ON DELETE CASCADE,
    UNIQUE (character_id, corporation_id)
);

CREATE INDEX character_loyalty_point_entries_idx1 ON character_loyalty_point_entries (character_id);

CREATE INDEX character_loyalty_point_entries_idx2 ON character_loyalty_point_entries (corporation_id);
