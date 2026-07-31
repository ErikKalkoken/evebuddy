CREATE TABLE eve_bloodlines (
    id INTEGER PRIMARY KEY,
    charisma INTEGER,
    corporation_id INTEGER NOT NULL,
    description TEXT NOT NULL,
    intelligence INTEGER,
    memory INTEGER,
    name TEXT NOT NULL,
    perception INTEGER,
    race_id INTEGER NOT NULL,
    ship_type_id INTEGER,
    willpower INTEGER,
    FOREIGN KEY (corporation_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    FOREIGN KEY (race_id) REFERENCES eve_races (id) ON DELETE CASCADE,
    FOREIGN KEY (ship_type_id) REFERENCES eve_types (id) ON DELETE SET NULL
);

CREATE INDEX eve_bloodlines_idx1 ON eve_bloodlines (corporation_id);

CREATE INDEX eve_bloodlines_idx2 ON eve_bloodlines (race_id);

CREATE INDEX eve_bloodlines_idx3 ON eve_bloodlines (ship_type_id);

ALTER TABLE eve_characters
ADD COLUMN bloodline_id INTEGER
REFERENCES eve_bloodlines(id) ON DELETE SET NULL;

CREATE INDEX eve_characters_idx5 ON eve_characters (bloodline_id);
