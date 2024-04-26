-- name: CreateMail :one
INSERT INTO mails (
    body,
    my_character_id,
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
INSERT INTO mail_recipients (
    mail_id,
    eve_entity_id
)
VALUES (?, ?);

-- name: CreateMailMailLabel :exec
INSERT INTO mail_mail_labels (
    mail_label_id,
    mail_id
)
VALUES (?, ?);

-- name: DeleteMail :exec
DELETE FROM mails
WHERE mails.my_character_id = ?
AND mails.mail_id = ?;

-- name: DeleteMailMailLabels :exec
DELETE FROM mail_mail_labels
WHERE mail_mail_labels.mail_id = ?;

-- name: GetMail :one
SELECT sqlc.embed(mails), sqlc.embed(eve_entities)
FROM mails
JOIN eve_entities ON eve_entities.id = mails.from_id
WHERE my_character_id = ?
AND mail_id = ?;

-- name: GetMailRecipients :many
SELECT eve_entities.*
FROM eve_entities
JOIN mail_recipients ON mail_recipients.eve_entity_id = eve_entities.id
WHERE mail_id = ?;

-- name: GetMailLabels :many
SELECT mail_labels.*
FROM mail_labels
JOIN mail_mail_labels ON mail_mail_labels.mail_label_id = mail_labels.id
WHERE mail_id = ?;

-- name: GetMailUnreadCount :one
SELECT COUNT(mails.id)
FROM mails
WHERE mails.my_character_id = ?
AND is_read IS FALSE;

-- name: GetMailLabelUnreadCounts :many
SELECT label_id, COUNT(mails.id) AS unread_count_2
FROM mail_labels
JOIN mail_mail_labels ON mail_mail_labels.mail_label_id = mail_labels.id
JOIN mails ON mails.id = mail_mail_labels.mail_id
WHERE mail_labels.my_character_id = ?
AND is_read IS FALSE
GROUP BY label_id;

-- name: GetMailListUnreadCounts :many
SELECT eve_entities.id AS list_id, COUNT(mails.id) as unread_count_2
FROM mails
JOIN mail_recipients ON mail_recipients.mail_id = mails.id
JOIN eve_entities ON eve_entities.id = mail_recipients.eve_entity_id
WHERE my_character_id = ?
AND eve_entities.category = "mail_list"
AND mails.is_read IS FALSE
GROUP BY eve_entities.id;

-- name: ListMailIDs :many
SELECT mail_id
FROM mails
WHERE my_character_id = ?;

-- name: ListMailIDsOrdered :many
SELECT mail_id
FROM mails
WHERE my_character_id = ?
ORDER BY timestamp DESC;

-- name: ListMailIDsNoLabelOrdered :many
SELECT mails.mail_id
FROM mails
LEFT JOIN mail_mail_labels ON mail_mail_labels.mail_id = mails.id
WHERE my_character_id = ?
AND mail_mail_labels.mail_id IS NULL
ORDER BY timestamp DESC;

-- name: ListMailIDsForLabelOrdered :many
SELECT mails.mail_id
FROM mails
JOIN mail_mail_labels ON mail_mail_labels.mail_id = mails.id
JOIN mail_labels ON mail_labels.id = mail_mail_labels.mail_label_id
WHERE mails.my_character_id = ?
AND label_id = ?
ORDER BY timestamp DESC;

-- name: ListMailIDsForListOrdered :many
SELECT mails.mail_id
FROM mails
JOIN mail_recipients ON mail_recipients.mail_id = mails.id
WHERE my_character_id = ?
AND mail_recipients.eve_entity_id = ?
ORDER BY timestamp DESC;

-- name: UpdateMail :exec
UPDATE mails
SET is_read = ?2
WHERE id = ?1;
