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
