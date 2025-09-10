CREATE TABLE corporation_structures (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    fuel_expires DATETIME,
    name TEXT NOT NULL,
    next_reinforce_apply DATETIME,
    next_reinforce_hour INTEGER,
    profile_id INTEGER NOT NULL,
    reinforce_hour INTEGER,
    state TEXT NOT NULL,
    state_timer_end DATETIME,
    state_timer_start DATETIME,
    structure_id INTEGER NOT NULL,
    system_id INTEGER NOT NULL,
    type_id INTEGER NOT NULL,
    unanchors_at DATETIME,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    FOREIGN KEY (system_id) REFERENCES eve_solar_systems (id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, structure_id)
);

CREATE INDEX corporation_structures_idx1 ON corporation_structures (corporation_id);

CREATE INDEX corporation_structures_idx2 ON corporation_structures (system_id);

CREATE INDEX corporation_structures_idx3 ON corporation_structures (type_id);

CREATE TABLE corporation_structure_services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_structure_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    state TEXT NOT NULL,
    FOREIGN KEY (corporation_structure_id) REFERENCES corporation_structures (id) ON DELETE CASCADE,
    UNIQUE (corporation_structure_id, name)
);

CREATE INDEX corporation_structure_services_idx1 ON corporation_structure_services (corporation_structure_id);
