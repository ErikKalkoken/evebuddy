CREATE TABLE eve_entities (
    id INTEGER PRIMARY KEY NOT NULL,
    category TEXT NOT NULL,
    name TEXT NOT NULL
);
CREATE INDEX eve_entities_name_idx ON eve_entities (name);
CREATE INDEX eve_entities_category_idx ON eve_entities (category);

CREATE TABLE races (
    id INTEGER PRIMARY KEY NOT NULL,
    description TEXT NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE characters (
    alliance_id INTEGER,
    birthday DATETIME NOT NULL,
    corporation_id INTEGER NOT NULL,
    description TEXT NOT NULL,
    gender TEXT NOT NULL,
    faction_id INTEGER,
    id INTEGER PRIMARY KEY NOT NULL,
    last_login_at DATETIME NOT NULL,
    location_id INTEGER NOT NULL,
    mail_updated_at DATETIME,
    name TEXT NOT NULL,
    race_id INTEGER NOT NULL,
    security_status REAL NOT NULL,
    skill_points INTEGER NOT NULL,
    wallet_balance REAL NOT NULL,
    FOREIGN KEY (alliance_id) REFERENCES eve_entities(id) ON DELETE SET NULL,
    FOREIGN KEY (corporation_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (faction_id) REFERENCES eve_entities(id) ON DELETE SET NULL,
    FOREIGN KEY (location_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (race_id) REFERENCES races(id) ON DELETE CASCADE
);

CREATE TABLE mail_lists (
    character_id INTEGER NOT NULL,
    eve_entity_id INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (character_id, eve_entity_id)
);

CREATE TABLE mail_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    color TEXT NOT NULL,
    label_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    unread_count INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    UNIQUE (character_id, label_id)
);

CREATE TABLE mail_recipients (
    mail_id INTEGER NOT NULL,
    eve_entity_id INTEGER NOT NULL,
    PRIMARY KEY (mail_id,eve_entity_id),
    FOREIGN KEY (mail_id) REFERENCES mails(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE
);

CREATE TABLE mails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    body TEXT NOT NULL,
    character_id INTEGER NOT NULL,
    from_id INTEGER NOT NULL,
    is_read BOOL NOT NULL,
    mail_id INTEGER NOT NULL,
    subject TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (from_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (character_id, mail_id)
);
CREATE INDEX mails_timestamp_idx ON mails (timestamp DESC);

CREATE TABLE mail_mail_labels (
    mail_label_id INTEGER NOT NULL,
    mail_id INTEGER NOT NULL,
    PRIMARY KEY (mail_label_id,mail_id),
    FOREIGN KEY (mail_label_id) REFERENCES mail_labels(id) ON DELETE CASCADE,
    FOREIGN KEY (mail_id) REFERENCES mails(id) ON DELETE CASCADE
);

CREATE TABLE tokens (
    access_token TEXT NOT NULL,
    character_id INTEGER PRIMARY KEY NOT NULL,
    expires_at DATETIME NOT NULL,
    refresh_token TEXT NOT NULL,
    token_type TEXT NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
);

CREATE TABLE dictionary (
    key TEXT PRIMARY KEY NOT NULL,
    value BLOB NOT NULL
);
