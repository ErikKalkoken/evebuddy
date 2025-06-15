CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    UNIQUE (name)
);

CREATE TABLE characters_tags (
    character_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE,
    UNIQUE (character_id, tag_id)
);

CREATE INDEX characters_tags_idx1 ON characters_tags (character_id);
CREATE INDEX characters_tags_idx2 ON characters_tags (tag_id);
