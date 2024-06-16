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
SELECT label_id, COUNT(cm.id) AS unread_count_2
FROM character_mail_labels cml
JOIN character_mail_mail_labels cmml ON cmml.character_mail_label_id = cml.id
JOIN character_mails cm ON cm.id = cmml.character_mail_id
WHERE cml.character_id = ?
AND is_read IS FALSE
GROUP BY label_id;

-- name: GetCharacterMailListUnreadCounts :many
SELECT eve_entities.id AS list_id, COUNT(cm.id) as unread_count_2
FROM character_mails cm
JOIN character_mails_recipients ON character_mails_recipients.mail_id = cm.id
JOIN eve_entities ON eve_entities.id = character_mails_recipients.eve_entity_id
WHERE character_id = ?
AND eve_entities.category = "mail_list"
AND cm.is_read IS FALSE
GROUP BY eve_entities.id;

-- name: ListMailIDs :many
SELECT mail_id
FROM character_mails
WHERE character_id = ?;

-- name: ListMailsOrdered :many
SELECT cm.subject, cm.mail_id, cm.timestamp, cm.is_read, ee.name as from_name
FROM character_mails cm
JOIN eve_entities ee ON ee.id = cm.from_id
WHERE character_id = ?
ORDER BY timestamp DESC;

-- name: ListMailsNoLabelOrdered :many
SELECT cm.subject, cm.mail_id, cm.timestamp, cm.is_read, ee.name as from_name
FROM character_mails cm
JOIN eve_entities ee ON ee.id = cm.from_id
LEFT JOIN character_mail_mail_labels cml ON cml.character_mail_id = cm.id
WHERE character_id = ?
AND cml.character_mail_id IS NULL
ORDER BY timestamp DESC;

-- name: ListMailsForLabelOrdered :many
SELECT cm.subject, cm.mail_id, cm.timestamp, cm.is_read, ee.name as from_name
FROM character_mails cm
JOIN eve_entities ee ON ee.id = cm.from_id
JOIN character_mail_mail_labels cml ON cml.character_mail_id = cm.id
JOIN character_mail_labels ON character_mail_labels.id = cml.character_mail_label_id
WHERE cm.character_id = ?
AND label_id = ?
ORDER BY timestamp DESC;

-- name: ListMailsForSentOrdered :many
SELECT cm.subject, cm.mail_id, cm.timestamp, cm.is_read, group_concat(ee.name, ", ") as from_name
FROM character_mails cm
JOIN character_mails_recipients cmr ON cmr.mail_id = cm.id
JOIN eve_entities ee ON ee.id = cmr.eve_entity_id
JOIN character_mail_mail_labels cml ON cml.character_mail_id = cm.id
JOIN character_mail_labels ON character_mail_labels.id = cml.character_mail_label_id
WHERE cm.character_id = ?
AND label_id = ?
GROUP BY cm.mail_id
ORDER BY timestamp DESC;

-- name: ListMailsForListOrdered :many
SELECT cm.subject, cm.mail_id, cm.timestamp, cm.is_read, ee.name as from_name
FROM character_mails cm
JOIN eve_entities ee ON ee.id = cm.from_id
JOIN character_mails_recipients cmr ON cmr.mail_id = cm.id
WHERE character_id = ?
AND cmr.eve_entity_id = ?
ORDER BY timestamp DESC;

-- name: UpdateMail :exec
UPDATE character_mails
SET is_read = ?2
WHERE id = ?1;
