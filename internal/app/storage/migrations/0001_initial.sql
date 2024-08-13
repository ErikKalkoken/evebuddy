-- common
CREATE TABLE general_section_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    section_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL,
    content_hash TEXT NOT NULL,
    completed_at DATETIME,
    error TEXT NOT NULL,
    started_at DATETIME,
    UNIQUE (section_id)
);

-- eve universe
CREATE TABLE eve_entities (
    id INTEGER PRIMARY KEY NOT NULL,
    category TEXT NOT NULL,
    name TEXT NOT NULL
);
CREATE INDEX eve_entities_idx1 ON eve_entities (name);
CREATE INDEX eve_entities_idx2 ON eve_entities (category);

CREATE TABLE eve_dogma_attributes (
    id INTEGER PRIMARY KEY NOT NULL,
    default_value REAL NOT NULL,
    description TEXT NOT NULL,
    display_name TEXT NOT NULL,
    icon_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    is_high_good BOOL NOT NULL,
    is_published BOOL NOT NULL,
    is_stackable BOOL NOT NULL,
    unit_id INTEGER NOT NULL
);

CREATE TABLE eve_categories (
    id INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    is_published BOOL NOT NULL
);
CREATE INDEX eve_categories_idx1 ON eve_categories (name);

CREATE TABLE eve_groups (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_category_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    is_published BOOL NOT NULL,
    FOREIGN KEY (eve_category_id) REFERENCES eve_categories(id) ON DELETE CASCADE
);
CREATE INDEX eve_groups_idx1 ON eve_groups (eve_category_id);
CREATE INDEX eve_groups_idx2 ON eve_groups (name);

CREATE TABLE eve_types (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_group_id INTEGER NOT NULL,
    capacity REAL NOT NULL,
    description TEXT NOT NULL,
    graphic_id INTEGER NOT NULL,
    icon_id INTEGER NOT NULL,
    is_published BOOL NOT NULL,
    market_group_id INTEGER NOT NULL,
    mass REAL NOT NULL,
    name TEXT NOT NULL,
    packaged_volume REAL NOT NULL,
    portion_size INTEGER NOT NULL,
    radius REAL NOT NULL,
    volume REAL NOT NULL,
    FOREIGN KEY (eve_group_id) REFERENCES eve_groups(id) ON DELETE CASCADE
);
CREATE INDEX eve_types_idx1 ON eve_types (eve_group_id);
CREATE INDEX eve_types_idx2 ON eve_types (name);

CREATE TABLE eve_type_dogma_attributes (
    id INTEGER PRIMARY KEY NOT NULL,
    dogma_attribute_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    value REAL NOT NULL,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    FOREIGN KEY (dogma_attribute_id) REFERENCES eve_dogma_attributes(id) ON DELETE CASCADE,
    UNIQUE (eve_type_id, dogma_attribute_id)
);
CREATE INDEX eve_type_dogma_attributes_idx1 ON eve_type_dogma_attributes (dogma_attribute_id);
CREATE INDEX eve_type_dogma_attributes_idx2 ON eve_type_dogma_attributes (eve_type_id);

CREATE TABLE eve_type_dogma_effects (
    id INTEGER PRIMARY KEY NOT NULL,
    dogma_effect_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    is_default BOOL NOT NULL,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (eve_type_id, dogma_effect_id)
);
CREATE INDEX eve_type_dogma_effects_idx1 ON eve_type_dogma_effects (dogma_effect_id);
CREATE INDEX eve_type_dogma_effects_idx2 ON eve_type_dogma_effects (eve_type_id);

CREATE TABLE eve_regions (
    id INTEGER PRIMARY KEY NOT NULL,
    description TEXT NOT NULL,
    name TEXT NOT NULL
);
CREATE INDEX eve_regions_idx1 ON eve_regions (name);

CREATE TABLE eve_constellations (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_region_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (eve_region_id) REFERENCES eve_regions(id) ON DELETE CASCADE
);
CREATE INDEX eve_constellations_idx1 ON eve_constellations (eve_region_id);
CREATE INDEX eve_constellations_idx2 ON eve_constellations (name);

CREATE TABLE eve_solar_systems (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_constellation_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    security_status REAL NOT NULL,
    FOREIGN KEY (eve_constellation_id) REFERENCES eve_constellations(id) ON DELETE CASCADE
);
CREATE INDEX eve_solar_systems_idx1 ON eve_solar_systems (eve_constellation_id);
CREATE INDEX eve_solar_systems_idx2 ON eve_solar_systems (name);

CREATE TABLE eve_races (
    id INTEGER PRIMARY KEY NOT NULL,
    description TEXT NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE eve_ship_skills (
    id INTEGER PRIMARY KEY NOT NULL,
    rank INTEGER NOT NULL,
    ship_type_id INTEGER NOT NULL,
    skill_type_id INTEGER NOT NULL,
    skill_level INTEGER NOT NULL,
    FOREIGN KEY (ship_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    FOREIGN KEY (skill_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (ship_type_id, rank)
);
CREATE INDEX eve_ship_skills_idx1 ON eve_ship_skills (rank);
CREATE INDEX eve_ship_skills_idx2 ON eve_ship_skills (ship_type_id);
CREATE INDEX eve_ship_skills_idx3 ON eve_ship_skills (skill_type_id);

CREATE TABLE eve_market_prices(
    type_id INTEGER PRIMARY KEY NOT NULL,
    adjusted_price REAL NOT NULL,
    average_price REAL NOT NULL
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
CREATE INDEX eve_characters_idx1 ON eve_characters (alliance_id);
CREATE INDEX eve_characters_idx2 ON eve_characters (corporation_id);
CREATE INDEX eve_characters_idx3 ON eve_characters (faction_id);
CREATE INDEX eve_characters_idx4 ON eve_characters (race_id);

CREATE VIEW eve_character_alliances AS
SELECT eve_entities.*
FROM eve_characters
LEFT JOIN eve_entities ON eve_entities.id = eve_characters.alliance_id;

CREATE VIEW eve_character_factions AS
SELECT eve_entities.*
FROM eve_characters
LEFT JOIN eve_entities ON eve_entities.id = eve_characters.faction_id;

CREATE TABLE eve_locations (
    id INTEGER PRIMARY KEY NOT NULL,
    eve_solar_system_id INTEGER,
    eve_type_id INTEGER,
    name TEXT NOT NULL,
    owner_id INTEGER,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (eve_solar_system_id) REFERENCES eve_solar_systems(id) ON DELETE SET NULL,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE SET NULL,
    FOREIGN KEY (owner_id) REFERENCES eve_entities(id) ON DELETE SET NULL
);
CREATE INDEX eve_locations_idx1 ON eve_locations (eve_solar_system_id);
CREATE INDEX eve_locations_idx2 ON eve_locations (eve_type_id);
CREATE INDEX eve_locations_idx3 ON eve_locations (owner_id);
CREATE INDEX eve_locations_idx4 ON eve_locations (name);

-- characters
CREATE TABLE characters (
    id INTEGER PRIMARY KEY NOT NULL,
    asset_value REAL,
    home_id INTEGER,
    last_login_at DATETIME,
    location_id INTEGER,
    ship_id INTEGER,
    total_sp INTEGER,
    unallocated_sp INTEGER,
    wallet_balance REAL,
    FOREIGN KEY (id) REFERENCES eve_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (home_id) REFERENCES eve_locations(id) ON DELETE SET NULL,
    FOREIGN KEY (location_id) REFERENCES eve_locations(id) ON DELETE SET NULL,
    FOREIGN KEY (ship_id) REFERENCES eve_types(id) ON DELETE SET NULL
);
CREATE INDEX characters_idx1 ON characters (home_id);
CREATE INDEX characters_idx2 ON characters (location_id);
CREATE INDEX characters_idx3 ON characters (ship_id);

CREATE TABLE character_section_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    section_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL,
    content_hash TEXT NOT NULL,
    completed_at DATETIME,
    error TEXT NOT NULL,
    started_at DATETIME,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    UNIQUE (character_id, section_id)
);
CREATE INDEX character_section_status_idx1 ON character_section_status (character_id);

CREATE TABLE character_assets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    is_blueprint_copy BOOL NOT NULL,
    is_singleton BOOL NOT NULL,
    item_id INTEGER NOT NULL,
    location_flag TEXT NOT NULL,
    location_id INTEGER NOT NULL,
    location_type TEXT NOT NULL,
    name TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (character_id, item_id)
);
CREATE INDEX character_assets_idx1 ON character_assets (character_id);
CREATE INDEX character_assets_idx2 ON character_assets (item_id);
CREATE INDEX character_assets_idx3 ON character_assets (location_id);
CREATE INDEX character_assets_idx4 ON character_assets (name);

CREATE TABLE character_attributes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bonus_remaps INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    charisma INTEGER NOT NULL,
    intelligence INTEGER NOT NULL,
    last_remap_date DATETIME,
    memory INTEGER NOT NULL,
    perception INTEGER NOT NULL,
    willpower INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    UNIQUE (character_id)
);

CREATE TABLE character_jump_clones (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    jump_clone_id INTEGER NOT NULL,
    location_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES eve_locations(id) ON DELETE CASCADE,
    UNIQUE (character_id, jump_clone_id)
);
CREATE INDEX character_jump_clones_idx1 ON character_jump_clones (character_id);
CREATE INDEX character_jump_clones_idx2 ON character_jump_clones (location_id);

CREATE TABLE character_jump_clone_implants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    clone_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    FOREIGN KEY (clone_id) REFERENCES character_jump_clones(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (clone_id, eve_type_id)
);
CREATE INDEX character_jump_clone_implants_idx1 ON character_jump_clone_implants (clone_id);
CREATE INDEX character_jump_clone_implants_idx2 ON character_jump_clone_implants (eve_type_id);

CREATE TABLE character_implants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (character_id, eve_type_id)
);
CREATE INDEX character_implants_idx1 ON character_implants (character_id);
CREATE INDEX character_implants_idx2 ON character_implants (eve_type_id);

CREATE TABLE character_mail_lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    eve_entity_id INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (character_id, eve_entity_id)
);
CREATE INDEX character_mail_lists_idx1 ON character_mail_lists (character_id);
CREATE INDEX character_mail_lists_idx2 ON character_mail_lists (eve_entity_id);

CREATE TABLE character_mail_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    color TEXT NOT NULL,
    label_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    unread_count INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    UNIQUE (character_id, label_id)
);
CREATE INDEX character_mail_labels_idx1 ON character_mail_labels (character_id);
CREATE INDEX character_mail_labels_idx2 ON character_mail_labels (label_id);

CREATE TABLE character_mails_recipients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mail_id INTEGER NOT NULL,
    eve_entity_id INTEGER NOT NULL,
    FOREIGN KEY (mail_id) REFERENCES character_mails(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_entity_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (mail_id, eve_entity_id)
);
CREATE INDEX character_mails_recipients_idx1 ON character_mails_recipients (mail_id);
CREATE INDEX character_mails_recipients_idx2 ON character_mails_recipients (eve_entity_id);

CREATE TABLE character_mails (
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
CREATE INDEX character_mails_idx1 ON character_mails (character_id);
CREATE INDEX character_mails_idx2 ON character_mails (from_id);
CREATE INDEX character_mails_idx3 ON character_mails (timestamp DESC);

CREATE TABLE character_mail_mail_labels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_mail_label_id INTEGER NOT NULL,
    character_mail_id INTEGER NOT NULL,
    FOREIGN KEY (character_mail_label_id) REFERENCES character_mail_labels(id) ON DELETE CASCADE,
    FOREIGN KEY (character_mail_id) REFERENCES character_mails(id) ON DELETE CASCADE,
    UNIQUE (character_mail_label_id, character_mail_id)
);
CREATE INDEX character_mail_mail_labels_idx1 ON character_mail_mail_labels (character_mail_label_id);
CREATE INDEX character_mail_mail_labels_idx2 ON character_mail_mail_labels (character_mail_id);

CREATE TABLE character_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    body TEXT,
    character_id INTEGER NOT NULL,
    is_read BOOL NOT NULL,
    notification_id INTEGER NOT NULL,
    sender_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    title TEXT,
    type_id INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES notification_types(id) ON DELETE CASCADE,
    UNIQUE (character_id, notification_id)
);

CREATE TABLE notification_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    UNIQUE (name)
);

CREATE TABLE character_skills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    active_skill_level INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    skill_points_in_skill INTEGER NOT NULL,
    trained_skill_level INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (character_id, eve_type_id)
);
CREATE INDEX character_skills_idx1 ON character_skills (character_id);
CREATE INDEX character_skills_idx2 ON character_skills (eve_type_id);

CREATE TABLE character_skillqueue_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    finish_date DATETIME,
    finished_level INTEGER NOT NULL,
    level_end_sp INTEGER,
    level_start_sp INTEGER,
    queue_position INTEGER NOT NULL,
    start_date DATETIME,
    training_start_sp INTEGER,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (character_id, queue_position)
);
CREATE INDEX character_skillqueue_items_idx1 ON character_skillqueue_items (character_id);
CREATE INDEX character_skillqueue_items_idx2 ON character_skillqueue_items (eve_type_id);
CREATE INDEX character_skillqueue_items_idx3 ON character_skillqueue_items (queue_position);

CREATE TABLE scopes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    UNIQUE (name)
);

CREATE TABLE character_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    access_token TEXT NOT NULL,
    character_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    refresh_token TEXT NOT NULL,
    token_type TEXT NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    UNIQUE (character_id)
);

CREATE TABLE character_token_scopes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_token_id INTEGER NOT NULL,
    scope_id INTEGER NOT NULL,
    FOREIGN KEY (character_token_id) REFERENCES character_tokens(id) ON DELETE CASCADE,
    FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE,
    UNIQUE (character_token_id, scope_id)
);
CREATE INDEX character_token_scopes_idx1 ON character_token_scopes (character_token_id);
CREATE INDEX character_token_scopes_idx2 ON character_token_scopes (scope_id);

CREATE TABLE character_wallet_journal_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    amount REAL NOT NULL,
    balance REAL NOT NULL,
    character_id INTEGER NOT NULL,
    context_id INTEGER NOT NULL,
    context_id_type TEXT NOT NULL,
    date DATETIME NOT NULL,
    description TEXT NOT NULL,
    first_party_id INTEGER,
    ref_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    ref_type TEXT NOT NULL,
    second_party_id INTEGER,
    tax REAL NOT NULL,
    tax_receiver_id INTEGER,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (first_party_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (second_party_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (tax_receiver_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (character_id, id)
);
CREATE INDEX character_wallet_journal_entries_idx1 ON character_wallet_journal_entries (character_id);
CREATE INDEX character_wallet_journal_entries_idx2 ON character_wallet_journal_entries (date);
CREATE INDEX character_wallet_journal_entries_idx3 ON character_wallet_journal_entries (first_party_id);
CREATE INDEX character_wallet_journal_entries_idx4 ON character_wallet_journal_entries (second_party_id);
CREATE INDEX character_wallet_journal_entries_idx5 ON character_wallet_journal_entries (tax_receiver_id);

CREATE VIEW character_wallet_journal_entry_first_parties AS
SELECT eve_entities.*
FROM character_wallet_journal_entries
LEFT JOIN eve_entities ON eve_entities.id = character_wallet_journal_entries.first_party_id;

CREATE VIEW character_wallet_journal_entry_second_parties AS
SELECT eve_entities.*
FROM character_wallet_journal_entries
LEFT JOIN eve_entities ON eve_entities.id = character_wallet_journal_entries.second_party_id;

CREATE VIEW character_wallet_journal_entry_tax_receivers AS
SELECT eve_entities.*
FROM character_wallet_journal_entries
LEFT JOIN eve_entities ON eve_entities.id = character_wallet_journal_entries.tax_receiver_id;

CREATE TABLE character_wallet_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    client_id INTEGER NOT NULL,
    date DATETIME NOT NULL,
    eve_type_id INTEGER NOT NULL,
    is_buy BOOL NOT NULL,
    is_personal BOOL NOT NULL,
    journal_ref_id INTEGER NOT NULL,
    location_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    transaction_id INTEGER NOT NULL,
    unit_price REAL NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (client_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES eve_locations(id) ON DELETE CASCADE,
    UNIQUE (character_id, transaction_id)
);
CREATE INDEX character_wallet_transactions_idx1 ON character_wallet_transactions (character_id);
CREATE INDEX character_wallet_transactions_idx2 ON character_wallet_transactions (client_id);
CREATE INDEX character_wallet_transactions_idx3 ON character_wallet_transactions (date);
CREATE INDEX character_wallet_transactions_idx4 ON character_wallet_transactions (eve_type_id);
CREATE INDEX character_wallet_transactions_idx5 ON character_wallet_transactions (location_id);
