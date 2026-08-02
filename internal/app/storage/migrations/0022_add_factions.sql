CREATE TABLE eve_factions (
    id INTEGER PRIMARY KEY,
    corporation_id INTEGER,
    description TEXT NOT NULL,
    is_unique BOOLEAN NOT NULL,
    militia_corporation_id INTEGER,
    name TEXT NOT NULL,
    size_factor REAL NOT NULL,
    solar_system_id INTEGER,
    station_count INTEGER NOT NULL,
    station_system_count INTEGER NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (militia_corporation_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (solar_system_id) REFERENCES eve_solar_systems (id) ON DELETE SET NULL
);

CREATE INDEX eve_factions_idx1 ON eve_factions (corporation_id);

CREATE INDEX eve_factions_idx2 ON eve_factions (militia_corporation_id);

CREATE INDEX eve_factions_idx3 ON eve_factions (solar_system_id);
