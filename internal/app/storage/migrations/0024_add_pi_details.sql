ALTER TABLE planet_pins
ADD COLUMN extractor_cycle_time INTEGER;

ALTER TABLE planet_pins
ADD COLUMN extractor_head_radius REAL;

ALTER TABLE planet_pins
ADD COLUMN extractor_num_heads INTEGER;

ALTER TABLE planet_pins
ADD COLUMN extractor_qty_per_cycle INTEGER;

CREATE TABLE planet_pin_contents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    planet_pin_id INTEGER NOT NULL,
    type_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    FOREIGN KEY (planet_pin_id) REFERENCES planet_pins (id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    UNIQUE (planet_pin_id, type_id)
);

CREATE INDEX planet_pin_contents_idx1 ON planet_pin_contents (planet_pin_id);

CREATE INDEX planet_pin_contents_idx2 ON planet_pin_contents (type_id);

CREATE TABLE planet_routes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_planet_id INTEGER NOT NULL,
    content_type_id INTEGER NOT NULL,
    destination_pin_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    route_id INTEGER NOT NULL,
    source_pin_id INTEGER NOT NULL,
    FOREIGN KEY (character_planet_id) REFERENCES character_planets (id) ON DELETE CASCADE,
    FOREIGN KEY (content_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    UNIQUE (character_planet_id, route_id)
);

CREATE INDEX planet_routes_idx1 ON planet_routes (character_planet_id);

CREATE INDEX planet_routes_idx2 ON planet_routes (content_type_id);
