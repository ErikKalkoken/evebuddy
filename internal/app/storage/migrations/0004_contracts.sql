CREATE TABLE character_contracts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    acceptor_id INTEGER,
    assignee_id INTEGER,
    availability TEXT NOT NULL,
    buyout REAL NOT NULL,
    character_id INTEGER NOT NULL,
    collateral REAL NOT NULL,
    contract_id INTEGER NOT NULL,
    date_accepted DATETIME,
    date_completed DATETIME,
    date_expired DATETIME NOT NULL,
    date_issued DATETIME NOT NULL,
    days_to_complete INTEGER NOT NULL,
    end_location_id INTEGER,
    for_corporation BOOLEAN NOT NULL,
    issuer_corporation_id INTEGER NOT NULL,
    issuer_id INTEGER NOT NULL,
    price REAL NOT NULL,
    reward REAL NOT NULL,
    start_location_id INTEGER,
    status TEXT NOT NULL,
    status_notified TEXT NOT NULL,
    title TEXT NOT NULL,
    type TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    volume REAL NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (acceptor_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (assignee_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (issuer_corporation_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (issuer_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (end_location_id) REFERENCES eve_locations(id) ON DELETE CASCADE,
    FOREIGN KEY (start_location_id) REFERENCES eve_locations(id) ON DELETE CASCADE,
    UNIQUE (character_id, contract_id)
);

CREATE INDEX character_contracts_idx1 ON character_contracts (character_id);

CREATE INDEX character_contracts_idx2 ON character_contracts (acceptor_id);

CREATE INDEX character_contracts_idx3 ON character_contracts (assignee_id);

CREATE INDEX character_contracts_idx4 ON character_contracts (issuer_corporation_id);

CREATE INDEX character_contracts_idx5 ON character_contracts (issuer_id);

CREATE INDEX character_contracts_idx6 ON character_contracts (end_location_id);

CREATE INDEX character_contracts_idx7 ON character_contracts (start_location_id);

CREATE TABLE character_contract_bids (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contract_id INTEGER NOT NULL,
    amount REAL NOT NULL,
    bid_id INTEGER NOT NULL,
    bidder_id INTEGER NOT NULL,
    date_bid DATETIME NOT NULL,
    FOREIGN KEY (contract_id) REFERENCES character_contracts(id) ON DELETE CASCADE,
    FOREIGN KEY (bidder_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    UNIQUE (contract_id, bid_id)
);

CREATE INDEX character_contract_bids_idx1 ON character_contract_bids (contract_id);

CREATE INDEX character_contract_bids_idx2 ON character_contract_bids (bidder_id);

CREATE TABLE character_contract_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contract_id INTEGER NOT NULL,
    is_included BOOLEAN NOT NULL,
    is_singleton BOOLEAN NOT NULL,
    quantity INTEGER NOT NULL,
    raw_quantity INTEGER NOT NULL,
    record_id INTEGER NOT NULL,
    type_id INTEGER NOT NULL,
    FOREIGN KEY (contract_id) REFERENCES character_contracts(id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES eve_types(id) ON DELETE CASCADE,
    UNIQUE (contract_id, record_id)
);

CREATE INDEX character_contract_items_idx1 ON character_contract_items (contract_id);

CREATE INDEX character_contract_items_idx2 ON character_contract_items (type_id);

CREATE INDEX character_contract_items_idx3 ON character_contract_items (record_id);