CREATE TABLE character_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    UNIQUE (name)
);

CREATE TABLE characters_character_tags (
    character_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES character_tags (id) ON DELETE CASCADE,
    UNIQUE (character_id, tag_id)
);

CREATE INDEX characters_character_tags_idx1 ON characters_character_tags (character_id);
CREATE INDEX characters_character_tags_idx2 ON characters_character_tags (tag_id);
