CREATE TABLE character_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL,
    is_read BOOL NOT NULL,
    notification_id INTEGER NOT NULL,
    sender_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    type_id INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES eve_entities(id) ON DELETE CASCADE,
    FOREIGN KEY (type_id) REFERENCES notification_types(id) ON DELETE CASCADE,
    UNIQUE (character_id, notification_id)
);

CREATE TABLE notification_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    UNIQUE (name)
);
