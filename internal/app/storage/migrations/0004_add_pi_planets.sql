CREATE TABLE planet_pins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_planet_id INTEGER NOT NULL,
    expiry_time DATETIME,
    extractor_product_type_id INTEGER,
    factory_schema_id INTEGER,
    install_time DATETIME,
    last_cycle_start DATETIME,
    pin_id INTEGER NOT NULL,
    schematic_id INTEGER,
    type_id INTEGER NOT NULL,
    FOREIGN KEY (character_planet_id) REFERENCES character_planets(id) ON DELETE CASCADE,
    FOREIGN KEY (extractor_product_type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    FOREIGN KEY (factory_schema_id) REFERENCES eve_schematics(id) ON DELETE CASCADE,
    FOREIGN KEY (schematic_id) REFERENCES eve_schematics(id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (character_planet_id, pin_id)
);

CREATE INDEX planet_pins_idx1 ON planet_pins (character_planet_id);

CREATE INDEX planet_pins_idx2 ON planet_pins (extractor_product_type_id);

CREATE INDEX planet_pins_idx3 ON planet_pins (factory_schema_id);

CREATE INDEX planet_pins_idx4 ON planet_pins (type_id);

CREATE INDEX planet_pins_idx5 ON planet_pins (pin_id);

CREATE INDEX planet_pins_idx6 ON planet_pins (schematic_id);
