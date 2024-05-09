CREATE TABLE dictionary (
    key TEXT PRIMARY KEY NOT NULL,
    value BLOB NOT NULL
);


CREATE TABLE eve_entities (
    id INTEGER PRIMARY KEY NOT NULL,
    category TEXT NOT NULL,
    name TEXT NOT NULL
);
CREATE INDEX eve_entities_name_idx ON eve_entities (name);
CREATE INDEX eve_entities_category_idx ON eve_entities (category);

CREATE TABLE eve_races (
    id INTEGER PRIMARY KEY NOT NULL,
    description TEXT NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE eve_categories (
    id INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    is_published BOOL NOT NULL
);

CREATE TABLE eve_groups (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    is_published BOOL NOT NULL,
    FOREIGN KEY (eve_category_id) REFERENCES eve_categories(id) ON DELETE CASCADE
);

CREATE TABLE eve_types (
    id INTEGER PRIMARY KEY NOT NULL,
    description TEXT NOT NULL,
    eve_group_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    is_published BOOL NOT NULL,
    FOREIGN KEY (eve_group_id) REFERENCES eve_groups(id) ON DELETE CASCADE
);

CREATE TABLE eve_regions (
    id INTEGER PRIMARY KEY NOT NULL,
    description TEXT NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE eve_constellations (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_region_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (eve_region_id) REFERENCES eve_regions(id) ON DELETE CASCADE
);

CREATE TABLE eve_solar_systems (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_constellation_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    security_status REAL NOT NULL,
    FOREIGN KEY (eve_constellation_id) REFERENCES eve_constellations(id) ON DELETE CASCADE
);

CREATE TABLE eve_characters (
    alliance_id INTEGER,
    birthday DATETIME NOT NULL,
    corporation_id INTEGER NOT NULL,
    description TEXT NOT NULL,
    gender TEXT NOT NULL,
    faction_id INTEGER,
    id INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    race_id INTEGER NOT NULL,
    security_status REAL NOT NULL,
    title TEXT NOT NULL,
    FOREIGN KEY (alliance_id) REFERENCES eve_entities(id) ON DELETE SET NULL,
    FOREIGN KEY (corporation_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (faction_id) REFERENCES eve_entities(id) ON DELETE SET NULL,
    FOREIGN KEY (race_id) REFERENCES eve_races(id) ON DELETE CASCADE
);

CREATE VIEW eve_character_alliances AS
SELECT eve_entities.*
FROM eve_characters
LEFT JOIN eve_entities ON eve_entities.id = eve_characters.alliance_id;

CREATE VIEW eve_character_factions AS
SELECT eve_entities.*
FROM eve_characters
LEFT JOIN eve_entities ON eve_entities.id = eve_characters.faction_id;

CREATE TABLE my_characters (
    id INTEGER PRIMARY KEY NOT NULL,
    last_login_at DATETIME NOT NULL,
    location_id INTEGER NOT NULL,
    ship_id INTEGER NOT NULL,
    skill_points INTEGER NOT NULL,
    wallet_balance REAL NOT NULL,
    FOREIGN KEY (id) REFERENCES eve_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES eve_solar_systems(id) ON DELETE CASCADE,
    FOREIGN KEY (ship_id) REFERENCES eve_types(id) ON DELETE CASCADE
);

CREATE TABLE mail_lists (
    my_character_id INTEGER NOT NULL,
    eve_entity_id INTEGER NOT NULL,
    FOREIGN KEY (my_character_id) REFERENCES my_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (my_character_id, eve_entity_id)
);

CREATE TABLE mail_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    my_character_id INTEGER NOT NULL,
    color TEXT NOT NULL,
    label_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    unread_count INTEGER NOT NULL,
    FOREIGN KEY (my_character_id) REFERENCES my_characters(id) ON DELETE CASCADE,
    UNIQUE (my_character_id, label_id)
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
    my_character_id INTEGER NOT NULL,
    from_id INTEGER NOT NULL,
    is_read BOOL NOT NULL,
    mail_id INTEGER NOT NULL,
    subject TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    FOREIGN KEY (my_character_id) REFERENCES my_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (from_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (my_character_id, mail_id)
);
CREATE INDEX mails_timestamp_idx ON mails (timestamp DESC);

CREATE TABLE mail_mail_labels (
    mail_label_id INTEGER NOT NULL,
    mail_id INTEGER NOT NULL,
    PRIMARY KEY (mail_label_id,mail_id),
    FOREIGN KEY (mail_label_id) REFERENCES mail_labels(id) ON DELETE CASCADE,
    FOREIGN KEY (mail_id) REFERENCES mails(id) ON DELETE CASCADE
);

CREATE TABLE skillqueue_items (
    eve_type_id INTEGER NOT NULL,
    finish_date DATETIME,
    finished_level INTEGER NOT NULL,
    level_end_sp INTEGER,
    level_start_sp INTEGER,
    queue_position INTEGER NOT NULL,
    my_character_id INTEGER NOT NULL,
    start_date DATETIME,
    training_start_sp INTEGER,
    FOREIGN KEY (my_character_id) REFERENCES my_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (my_character_id, queue_position)
);
CREATE INDEX skillqueue_items_queue_position_idx ON skillqueue_items (queue_position ASC);


CREATE TABLE scopes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    UNIQUE (name)
);
CREATE INDEX scopes_name_idx ON scopes (name ASC);

CREATE TABLE tokens (
    access_token TEXT NOT NULL,
    my_character_id INTEGER PRIMARY KEY NOT NULL,
    expires_at DATETIME NOT NULL,
    refresh_token TEXT NOT NULL,
    token_type TEXT NOT NULL,
    FOREIGN KEY (my_character_id) REFERENCES my_characters(id) ON DELETE CASCADE
);

CREATE TABLE tokens_scopes (
    token_id INTEGER NOT NULL,
    scope_id INTEGER NOT NULL,
    PRIMARY KEY (token_id,scope_id),
    FOREIGN KEY (token_id) REFERENCES tokens(my_character_id) ON DELETE CASCADE,
    FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE
);

CREATE TABLE wallet_journal_entries (
    amount REAL NOT NULL,
    balance REAL NOT NULL,
    context_id INTEGER NOT NULL,
    context_id_type TEXT NOT NULL,
    date DATETIME NOT NULL,
    description TEXT NOT NULL,
    first_party_id INTEGER,
    id INTEGER NOT NULL,
    my_character_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    ref_type TEXT NOT NULL,
    second_party_id INTEGER,
    tax REAL NOT NULL,
    tax_receiver_id INTEGER,
    FOREIGN KEY (first_party_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (my_character_id) REFERENCES my_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (second_party_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (tax_receiver_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (my_character_id, id)
);
CREATE INDEX wallet_journal_entries_date_idx ON wallet_journal_entries (date ASC);

CREATE VIEW wallet_journal_entry_first_parties AS
SELECT eve_entities.*
FROM wallet_journal_entries
LEFT JOIN eve_entities ON eve_entities.id = wallet_journal_entries.first_party_id;

CREATE VIEW wallet_journal_entry_second_parties AS
SELECT eve_entities.*
FROM wallet_journal_entries
LEFT JOIN eve_entities ON eve_entities.id = wallet_journal_entries.second_party_id;

CREATE VIEW wallet_journal_entry_tax_receivers AS
SELECT eve_entities.*
FROM wallet_journal_entries
LEFT JOIN eve_entities ON eve_entities.id = wallet_journal_entries.tax_receiver_id;
