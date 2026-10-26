CREATE TABLE skill_plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE CHECK (length(trim(name)) > 0)
);

CREATE TABLE skill_plan_requirements (
    plan_id INTEGER NOT NULL REFERENCES skill_plans (id) ON DELETE CASCADE,
    type_id INTEGER NOT NULL REFERENCES eve_types (id),
    level INTEGER NOT NULL CHECK (level BETWEEN 1 AND 5),
    PRIMARY KEY (plan_id, type_id)
);
