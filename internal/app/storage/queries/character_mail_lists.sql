-- name: CreateCharacterMailList :exec
INSERT OR IGNORE INTO character_mail_lists (
    character_id,
    eve_entity_id
)
VALUES (
    ?, ?
);

-- name: DeleteObsoleteCharacterMailLists :exec
DELETE FROM character_mail_lists
WHERE character_mail_lists.character_id = ?
AND eve_entity_id NOT IN (
    SELECT eve_entity_id
    FROM character_mails_recipients
    JOIN character_mails ON character_mails.id = character_mails_recipients.mail_id
    WHERE character_mails.character_id = ?
)
AND eve_entity_id NOT IN (
    SELECT from_id
    FROM character_mails
    WHERE character_mails.character_id = ?
);

-- name: GetCharacterMailList :one
SELECT *
FROM character_mail_lists
WHERE character_id = ? AND eve_entity_id = ?;

-- name: ListCharacterMailListsOrdered :many
SELECT eve_entities.*
FROM character_mail_lists
JOIN eve_entities ON eve_entities.id = character_mail_lists.eve_entity_id
WHERE character_id = ?
ORDER by eve_entities.name;