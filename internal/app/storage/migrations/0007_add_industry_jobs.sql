CREATE TABLE character_industry_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    activity_id INTEGER NOT NULL,
    blueprint_id INTEGER NOT NULL,
    blueprint_location_id INTEGER NOT NULL,
    blueprint_type_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
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
    station_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    successful_runs INTEGER,
    FOREIGN KEY (blueprint_location_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
    FOREIGN KEY (blueprint_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    FOREIGN KEY (completed_character_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
    FOREIGN KEY (installer_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    FOREIGN KEY (output_location_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
    FOREIGN KEY (product_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    FOREIGN KEY (station_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
    UNIQUE (character_id, job_id)
);

CREATE INDEX character_industry_jobs_idx1 ON character_industry_jobs (blueprint_location_id);

CREATE INDEX character_industry_jobs_idx2 ON character_industry_jobs (blueprint_type_id);

CREATE INDEX character_industry_jobs_idx3 ON character_industry_jobs (character_id);

CREATE INDEX character_industry_jobs_idx4 ON character_industry_jobs (completed_character_id);

CREATE INDEX character_industry_jobs_idx5 ON character_industry_jobs (installer_id);

CREATE INDEX character_industry_jobs_idx6 ON character_industry_jobs (output_location_id);

CREATE INDEX character_industry_jobs_idx7 ON character_industry_jobs (product_type_id);

CREATE INDEX character_industry_jobs_idx8 ON character_industry_jobs (station_id);

CREATE INDEX character_industry_jobs_idx9 ON character_industry_jobs (start_date DESC);