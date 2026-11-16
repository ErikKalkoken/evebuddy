CREATE TABLE skill_sets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE
);

CREATE TABLE skill_set_requirements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_set_id INTEGER NOT NULL,
    type_id INTEGER NOT NULL,
    level INTEGER NOT NULL,
    FOREIGN KEY (skill_set_id) REFERENCES skill_sets (id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES eve_types (id) ON DELETE CASCADE,
    UNIQUE (skill_set_id, type_id)
);

CREATE INDEX skill_set_requirements_idx1 ON skill_set_requirements (skill_set_id);

CREATE INDEX skill_set_requirements_idx2 ON skill_set_requirements (type_id);
