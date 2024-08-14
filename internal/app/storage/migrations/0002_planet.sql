CREATE TABLE eve_planets (
    id INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    eve_solar_system_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    FOREIGN KEY (eve_solar_system_id) REFERENCES eve_solar_systems(id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types(id) ON DELETE CASCADE
);
