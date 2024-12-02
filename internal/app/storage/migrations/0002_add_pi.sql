CREATE TABLE character_planets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    eve_planet_id INTEGER NOT NULL,
    last_update DATETIME NOT NULL,
    num_pins INTEGER NOT NULL,
    upgrade_level INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_planet_id) REFERENCES eve_planets(id) ON DELETE CASCADE,
    UNIQUE (character_id, eve_planet_id)
);

CREATE INDEX character_planets_idx1 ON character_planets (character_id);

CREATE INDEX character_planets_idx2 ON character_planets (eve_planet_id);
