CREATE TABLE eve_schematics (
    id INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    cycle_time INTEGER NOT NULL,
    UNIQUE(name)
);
