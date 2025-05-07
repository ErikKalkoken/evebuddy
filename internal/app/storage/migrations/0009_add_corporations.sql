CREATE TABLE eve_corporations (
    id INTEGER PRIMARY KEY NOT NULL,
    alliance_id INTEGER,
    ceo_id INTEGER,
    creator_id INTEGER,
    date_founded DATETIME,
    description TEXT NOT NULL,
    faction_id INTEGER,
    home_station_id INTEGER,
    member_count INTEGER NOT NULL,
    name TEXT NOT NULL,
    shares INTEGER,
    tax_rate REAL NOT NULL,
    ticker TEXT NOT NULL,
    url TEXT NOT NULL,
    war_eligible BOOL NOT NULL,
    FOREIGN KEY (alliance_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (ceo_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (creator_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (faction_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (home_station_id) REFERENCES eve_entities (id) ON DELETE SET NULL
);

CREATE INDEX eve_corporations_idx1 ON eve_corporations (alliance_id);

CREATE INDEX eve_corporations_idx2 ON eve_corporations (ceo_id);

CREATE INDEX eve_corporations_idx3 ON eve_corporations (creator_id);

CREATE INDEX eve_corporations_idx4 ON eve_corporations (faction_id);

CREATE INDEX eve_corporations_idx5 ON eve_corporations (home_station_id);

CREATE TABLE corporations (
    id INTEGER PRIMARY KEY NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id) REFERENCES eve_corporations (id) ON DELETE CASCADE
);

CREATE TABLE corporation_section_status (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    comment TEXT NOT NULL,
    corporation_id INTEGER NOT NULL,
    section_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL,
    content_hash TEXT NOT NULL,
    completed_at DATETIME,
    error TEXT NOT NULL,
    started_at DATETIME,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, section_id)
);

CREATE INDEX corporation_section_status_idx1 ON corporation_section_status (corporation_id);

CREATE INDEX corporation_section_status_idx2 ON corporation_section_status (section_id);

CREATE TABLE corporation_industry_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_id INTEGER NOT NULL,
    blueprint_id INTEGER NOT NULL,
    blueprint_location_id INTEGER NOT NULL,
    blueprint_type_id INTEGER NOT NULL,
    corporation_id INTEGER NOT NULL,
    completed_character_id INTEGER,
    completed_date DATETIME,
    cost REAL,
    duration INTEGER NOT NULL,
    end_date DATETIME NOT NULL,
    facility_id INTEGER NOT NULL,
    installer_id INTEGER NOT NULL,
    job_id INTEGER NOT NULL,
    licensed_runs INTEGER,
    output_location_id INTEGER NOT NULL,
    pause_date DATETIME,
    probability REAL,
    product_type_id INTEGER,
    runs INTEGER NOT NULL,
    start_date DATETIME NOT NULL,
    location_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    successful_runs INTEGER,
    FOREIGN KEY (blueprint_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    FOREIGN KEY (completed_character_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (installer_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    FOREIGN KEY (product_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, job_id)
);

CREATE INDEX corporation_industry_jobs_idx2 ON corporation_industry_jobs (blueprint_type_id);

CREATE INDEX corporation_industry_jobs_idx3 ON corporation_industry_jobs (corporation_id);

CREATE INDEX corporation_industry_jobs_idx4 ON corporation_industry_jobs (completed_character_id);

CREATE INDEX corporation_industry_jobs_idx6 ON corporation_industry_jobs (installer_id);

CREATE INDEX corporation_industry_jobs_idx8 ON corporation_industry_jobs (product_type_id);

CREATE INDEX corporation_industry_jobs_idx9 ON corporation_industry_jobs (location_id);

CREATE INDEX corporation_industry_jobs_idx10 ON corporation_industry_jobs (end_date DESC);

CREATE INDEX character_roles_jobs_idx2 ON character_roles (name);
