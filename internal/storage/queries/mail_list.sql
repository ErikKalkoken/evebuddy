-- name: CreateMailList :exec
INSERT OR IGNORE INTO mail_lists (
    character_id,
    eve_entity_id
)
VALUES (
    ?, ?
);

-- name: DeleteObsoleteMailLists :exec
DELETE FROM mail_lists
WHERE mail_lists.character_id = ?
AND eve_entity_id NOT IN (
    SELECT eve_entity_id
    FROM mail_recipients
    JOIN mails ON mails.id = mail_recipients.mail_id
    WHERE mails.character_id = ?
)
AND eve_entity_id NOT IN (
    SELECT from_id
    FROM mails
    WHERE mails.character_id = ?
);

-- name: GetMailList :one
SELECT *
FROM mail_lists
WHERE character_id = ? AND eve_entity_id = ?;

-- name: ListMailListsOrdered :many
SELECT eve_entities.*
FROM mail_lists
JOIN eve_entities ON eve_entities.id = mail_lists.eve_entity_id
WHERE character_id = ?
ORDER by eve_entities.name;