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

CREATE TABLE character_contact_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    label_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    UNIQUE (character_id, label_id)
);

CREATE INDEX character_contact_labels_idx1 ON character_contact_labels (character_id);

CREATE INDEX character_contact_labels_idx2 ON character_contact_labels (label_id);

CREATE TABLE character_contacts_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id INTEGER NOT NULL,
    label_id INTEGER NOT NULL,
    FOREIGN KEY (contact_id) REFERENCES character_contacts (id) ON DELETE CASCADE,
    FOREIGN KEY (label_id) REFERENCES character_contact_labels (id) ON DELETE CASCADE,
    UNIQUE (contact_id, label_id)
);

CREATE INDEX character_contacts_labels_idx1 ON character_contacts_labels (contact_id);

CREATE INDEX character_contacts_labels_idx2 ON character_contacts_labels (label_id);