ALTER TABLE character_notifications
ADD COLUMN recipient_id INTEGER REFERENCES eve_entities (id) ON DELETE SET NULL;

CREATE INDEX character_notifications_idx6 ON character_notifications (recipient_id);