CREATE TABLE character_market_orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    duration INTEGER NOT NULL,
    escrow REAL,
    is_buy_order BOOLEAN NOT NULL,
    is_corporation BOOLEAN NOT NULL,
    issued DATETIME NOT NULL,
    location_id INTEGER NOT NULL,
    min_volume INTEGER,
    order_id INTEGER NOT NULL,
    price REAL NOT NULL,
    range TEXT NOT NULL,
    region_id INTEGER NOT NULL,
    state TEXT NOT NULL,
    type_id INTEGER NOT NULL,
    volume_remains INTEGER NOT NULL,
    volume_total INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters (id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
    FOREIGN KEY (region_id) REFERENCES eve_regions (id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    UNIQUE (character_id, order_id)
);

CREATE INDEX character_market_orders_idx1 ON character_market_orders (character_id);
CREATE INDEX character_market_orders_idx2 ON character_market_orders (location_id);
CREATE INDEX character_market_orders_idx3 ON character_market_orders (region_id);
CREATE INDEX character_market_orders_idx4 ON character_market_orders (type_id);
