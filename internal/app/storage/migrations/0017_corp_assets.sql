CREATE TABLE corporation_assets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    is_blueprint_copy BOOL NOT NULL,
    is_singleton BOOL NOT NULL,
    item_id INTEGER NOT NULL,
    location_flag TEXT NOT NULL,
    location_id INTEGER NOT NULL,
    location_type TEXT NOT NULL,
    name TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, item_id)
);

CREATE INDEX corporation_assets_idx1 ON corporation_assets (corporation_id);

CREATE INDEX corporation_assets_idx2 ON corporation_assets (item_id);

CREATE INDEX corporation_assets_idx3 ON corporation_assets (location_id);

CREATE INDEX corporation_assets_idx4 ON corporation_assets (name);