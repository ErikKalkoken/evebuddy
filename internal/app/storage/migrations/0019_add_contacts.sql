CREATE TABLE character_contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    contact_id INTEGER NOT NULL,
    is_blocked BOOLEAN,
    is_watched BOOLEAN,
    standing REAL NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    FOREIGN KEY (contact_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    UNIQUE (character_id, contact_id)
);

CREATE INDEX character_contacts_idx1 ON character_contacts (character_id);

CREATE INDEX character_contacts_idx2 ON character_contacts (contact_id);
