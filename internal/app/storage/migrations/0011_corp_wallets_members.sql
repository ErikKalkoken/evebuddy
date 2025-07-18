CREATE TABLE corporation_hangar_names (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    division_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, division_id)
);

CREATE INDEX corporation_hangar_names_idx1 ON corporation_hangar_names (corporation_id);

CREATE TABLE corporation_wallet_balances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    division_id INTEGER NOT NULL,
    balance REAL NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, division_id)
);

CREATE INDEX corporation_wallet_balances_idx1 ON corporation_wallet_balances (corporation_id);

CREATE TABLE corporation_wallet_names (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    division_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, division_id)
);

CREATE INDEX corporation_wallet_names_idx1 ON corporation_wallet_names (corporation_id);

CREATE TABLE corporation_wallet_journal_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    amount REAL NOT NULL,
    balance REAL NOT NULL,
    corporation_id INTEGER NOT NULL,
    context_id INTEGER NOT NULL,
    context_id_type TEXT NOT NULL,
    date DATETIME NOT NULL,
    description TEXT NOT NULL,
    division_id INTEGER NOT NULL,
    first_party_id INTEGER,
    ref_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    ref_type TEXT NOT NULL,
    second_party_id INTEGER,
    tax REAL NOT NULL,
    tax_receiver_id INTEGER,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    FOREIGN KEY (first_party_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    FOREIGN KEY (second_party_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    FOREIGN KEY (tax_receiver_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, division_id, ref_id)
);

CREATE INDEX corporation_wallet_journal_entries_idx1 ON corporation_wallet_journal_entries (corporation_id);

CREATE INDEX corporation_wallet_journal_entries_idx2 ON corporation_wallet_journal_entries (division_id);

CREATE INDEX corporation_wallet_journal_entries_idx3 ON corporation_wallet_journal_entries (ref_id);

CREATE INDEX corporation_wallet_journal_entries_idx4 ON corporation_wallet_journal_entries (date DESC);

CREATE INDEX corporation_wallet_journal_entries_idx5 ON corporation_wallet_journal_entries (first_party_id);

CREATE INDEX corporation_wallet_journal_entries_idx6 ON corporation_wallet_journal_entries (second_party_id);

CREATE INDEX corporation_wallet_journal_entries_idx7 ON corporation_wallet_journal_entries (tax_receiver_id);

CREATE TABLE corporation_wallet_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    client_id INTEGER NOT NULL,
    date DATETIME NOT NULL,
    division_id INTEGER NOT NULL,
    eve_type_id INTEGER NOT NULL,
    is_buy BOOL NOT NULL,
    journal_ref_id INTEGER NOT NULL,
    location_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    transaction_id INTEGER NOT NULL,
    unit_price REAL NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    FOREIGN KEY (client_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    FOREIGN KEY (eve_type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    FOREIGN KEY (location_id) REFERENCES eve_locations (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, division_id, transaction_id)
);

CREATE INDEX corporation_wallet_transactions_idx1 ON corporation_wallet_transactions (corporation_id);

CREATE INDEX corporation_wallet_transactions_idx2 ON corporation_wallet_transactions (division_id);

CREATE INDEX corporation_wallet_transactions_idx3 ON corporation_wallet_transactions (transaction_id);

CREATE INDEX corporation_wallet_transactions_idx4 ON corporation_wallet_transactions (client_id);

CREATE INDEX corporation_wallet_transactions_idx5 ON corporation_wallet_transactions (date DESC);

CREATE INDEX corporation_wallet_transactions_idx6 ON corporation_wallet_transactions (eve_type_id);

CREATE INDEX corporation_wallet_transactions_idx7 ON corporation_wallet_transactions (location_id);

CREATE INDEX corporation_wallet_transactions_idx8 ON corporation_wallet_transactions (transaction_id);

CREATE TABLE corporation_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    corporation_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    FOREIGN KEY (corporation_id) REFERENCES corporations (id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES eve_entities (id) ON DELETE CASCADE,
    UNIQUE (corporation_id, character_id)
);

CREATE INDEX corporation_members_idx1 ON corporation_members (corporation_id);

CREATE INDEX corporation_members_idx2 ON corporation_members (character_id);
