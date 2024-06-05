-- name: CreateMail :one
INSERT INTO character_mails (
    body,
    character_id,
    from_id,
    is_read,
    mail_id,
    subject,
    timestamp
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: CreateMailRecipient :exec
INSERT INTO character_mails_recipients (
    mail_id,
    eve_entity_id
)
VALUES (?, ?);

-- name: CreateMailCharacterMailLabel :exec
INSERT INTO character_mail_mail_labels (
    character_mail_label_id,
    character_mail_id
)
VALUES (?, ?);

-- name: DeleteMail :exec
DELETE FROM character_mails
WHERE character_mails.character_id = ?
AND character_mails.mail_id = ?;

-- name: DeleteMailCharacterMailLabels :exec
DELETE FROM character_mail_mail_labels
WHERE character_mail_mail_labels.character_mail_id = ?;

-- name: GetMail :one
SELECT sqlc.embed(character_mails), sqlc.embed(eve_entities)
FROM character_mails
JOIN eve_entities ON eve_entities.id = character_mails.from_id
WHERE character_id = ?
AND mail_id = ?;

-- name: GetMailRecipients :many
SELECT eve_entities.*
FROM eve_entities
JOIN character_mails_recipients ON character_mails_recipients.eve_entity_id = eve_entities.id
WHERE mail_id = ?;

-- name: GetCharacterMailLabels :many
SELECT character_mail_labels.*
FROM character_mail_labels
JOIN character_mail_mail_labels ON character_mail_mail_labels.character_mail_label_id = character_mail_labels.id
WHERE character_mail_id = ?;

-- name: GetMailUnreadCount :one
SELECT COUNT(*)
FROM character_mails
WHERE character_mails.character_id = ?
AND is_read IS FALSE;

-- name: GetMailCount :one
SELECT COUNT(*)
FROM character_mails
WHERE character_mails.character_id = ?;

-- name: GetCharacterMailLabelUnreadCounts :many
SELECT label_id, COUNT(character_mails.id) AS unread_count_2
FROM character_mail_labels
JOIN character_mail_mail_labels ON character_mail_mail_labels.character_mail_label_id = character_mail_labels.id
JOIN character_mails ON character_mails.id = character_mail_mail_labels.character_mail_id
WHERE character_mail_labels.character_id = ?
AND is_read IS FALSE
GROUP BY label_id;

-- name: GetCharacterMailListUnreadCounts :many
SELECT eve_entities.id AS list_id, COUNT(character_mails.id) as unread_count_2
FROM character_mails
JOIN character_mails_recipients ON character_mails_recipients.mail_id = character_mails.id
JOIN eve_entities ON eve_entities.id = character_mails_recipients.eve_entity_id
WHERE character_id = ?
AND eve_entities.category = "mail_list"
AND character_mails.is_read IS FALSE
GROUP BY eve_entities.id;

-- name: ListMailIDs :many
SELECT mail_id
FROM character_mails
WHERE character_id = ?;

-- name: ListMailsOrdered :many
SELECT sqlc.embed(character_mails), sqlc.embed(eve_entities)
FROM character_mails
JOIN eve_entities ON eve_entities.id = character_mails.from_id
WHERE character_id = ?
ORDER BY timestamp DESC;

-- name: ListMailsNoLabelOrdered :many
SELECT sqlc.embed(character_mails), sqlc.embed(eve_entities)
FROM character_mails
JOIN eve_entities ON eve_entities.id = character_mails.from_id
LEFT JOIN character_mail_mail_labels ON character_mail_mail_labels.character_mail_id = character_mails.id
WHERE character_id = ?
AND character_mail_mail_labels.character_mail_id IS NULL
ORDER BY timestamp DESC;

-- name: ListMailsForLabelOrdered :many
SELECT sqlc.embed(character_mails), sqlc.embed(eve_entities)
FROM character_mails
JOIN eve_entities ON eve_entities.id = character_mails.from_id
JOIN character_mail_mail_labels ON character_mail_mail_labels.character_mail_id = character_mails.id
JOIN character_mail_labels ON character_mail_labels.id = character_mail_mail_labels.character_mail_label_id
WHERE character_mails.character_id = ?
AND label_id = ?
ORDER BY timestamp DESC;

-- name: ListMailsForListOrdered :many
SELECT sqlc.embed(character_mails), sqlc.embed(eve_entities)
FROM character_mails
JOIN eve_entities ON eve_entities.id = character_mails.from_id
JOIN character_mails_recipients ON character_mails_recipients.mail_id = character_mails.id
WHERE character_id = ?
AND character_mails_recipients.eve_entity_id = ?
ORDER BY timestamp DESC;

-- name: UpdateMail :exec
UPDATE character_mails
SET is_read = ?2
WHERE id = ?1;
