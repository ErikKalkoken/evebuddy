-- CREATE TABLE corporations (
--     id INTEGER PRIMARY KEY AUTOINCREMENT,
--     eve_corporation_id INTEGER NOT NULL,
--     FOREIGN KEY (eve_corporation_id) REFERENCES eve_corporations (id) ON DELETE CASCADE,
--     UNIQUE (eve_corporation_id)
-- )
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

-- CREATE TABLE corporation_industry_jobs (
--     id INTEGER PRIMARY KEY AUTOINCREMENT,
--     activity_id INTEGER NOT NULL,
--     blueprint_id INTEGER NOT NULL,
--     blueprint_location_id INTEGER NOT NULL,
--     blueprint_type_id INTEGER NOT NULL,
--     corporation_id INTEGER NOT NULL,
--     completed_character_id INTEGER,
--     completed_date DATETIME,
--     cost REAL,
--     duration INTEGER NOT NULL,
--     end_date DATETIME NOT NULL,
--     facility_id INTEGER NOT NULL,
--     installer_id INTEGER NOT NULL,
--     job_id INTEGER NOT NULL,
--     licensed_runs INTEGER,
--     output_location_id INTEGER NOT NULL,
--     pause_date DATETIME,
--     probability REAL,
--     product_type_id INTEGER,
--     runs INTEGER NOT NULL,
--     start_date DATETIME NOT NULL,
--     station_id INTEGER NOT NULL,
--     status TEXT NOT NULL,
--     successful_runs INTEGER,
--     FOREIGN KEY (blueprint_location_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
--     FOREIGN KEY (blueprint_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
--     FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
--     FOREIGN KEY (completed_character_id) REFERENCES eve_entities (id) ON DELETE SET NULL,
--     FOREIGN KEY (facility_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
--     FOREIGN KEY (installer_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
--     FOREIGN KEY (output_location_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
--     FOREIGN KEY (product_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
--     FOREIGN KEY (station_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
--     UNIQUE (corporation_id, job_id)
-- );
-- CREATE INDEX corporation_industry_jobs_idx1 ON corporation_industry_jobs (blueprint_location_id);
-- CREATE INDEX corporation_industry_jobs_idx2 ON corporation_industry_jobs (blueprint_type_id);
-- CREATE INDEX corporation_industry_jobs_idx3 ON corporation_industry_jobs (corporation_id);
-- CREATE INDEX corporation_industry_jobs_idx4 ON corporation_industry_jobs (completed_character_id);
-- CREATE INDEX corporation_industry_jobs_idx5 ON corporation_industry_jobs (facility_id);
-- CREATE INDEX corporation_industry_jobs_idx6 ON corporation_industry_jobs (installer_id);
-- CREATE INDEX corporation_industry_jobs_idx7 ON corporation_industry_jobs (output_location_id);
-- CREATE INDEX corporation_industry_jobs_idx8 ON corporation_industry_jobs (product_type_id);
-- CREATE INDEX corporation_industry_jobs_idx9 ON corporation_industry_jobs (station_id);
-- CREATE INDEX corporation_industry_jobs_idx10 ON corporation_industry_jobs (end_date DESC);