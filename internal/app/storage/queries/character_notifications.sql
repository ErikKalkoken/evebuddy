-- name: CountCharacterNotifications :many
SELECT
    cn.type_id,
    nt.name,
    SUM(NOT cn.is_read) AS unread_count,
    COUNT(*) AS total_count
FROM
    character_notifications cn
    JOIN notification_types nt ON nt.id = cn.type_id
WHERE
    character_id = ?
GROUP BY
    cn.type_id,
    nt.name;

-- name: CreateCharacterNotification :exec
INSERT INTO
    character_notifications (
        body,
        character_id,
        is_processed,
        is_read,
        notification_id,
        recipient_id,
        sender_id,
        TEXT,
        timestamp,
        title,
        type_id
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCharacterNotification :one
SELECT
    sqlc.embed(cn),
    sqlc.embed(sender),
    sqlc.embed(nt),
    recipient.name as recipient_name,
    recipient.category as recipient_category
FROM
    character_notifications cn
    JOIN eve_entities sender ON sender.id = cn.sender_id
    JOIN notification_types nt ON nt.id = cn.type_id
    LEFT JOIN eve_entities recipient ON recipient.id = cn.recipient_id
WHERE
    character_id = ?
    and notification_id = ?;

-- name: ListCharacterNotificationIDs :many
SELECT
    notification_id
FROM
    character_notifications
WHERE
    character_id = ?;

-- name: ListCharacterNotificationsTypes :many
SELECT
    sqlc.embed(cn),
    sqlc.embed(sender),
    sqlc.embed(nt),
    recipient.name as recipient_name,
    recipient.category as recipient_category
FROM
    character_notifications cn
    JOIN eve_entities sender ON sender.id = cn.sender_id
    JOIN notification_types nt ON nt.id = cn.type_id
    LEFT JOIN eve_entities recipient ON recipient.id = cn.recipient_id
WHERE
    character_id = ?
    AND nt.name IN (sqlc.slice('names'))
ORDER BY
    timestamp DESC;

-- name: ListCharacterNotificationsAll :many
SELECT
    sqlc.embed(cn),
    sqlc.embed(sender),
    sqlc.embed(nt),
    recipient.name as recipient_name,
    recipient.category as recipient_category
FROM
    character_notifications cn
    JOIN eve_entities sender ON sender.id = cn.sender_id
    JOIN notification_types nt ON nt.id = cn.type_id
    LEFT JOIN eve_entities recipient ON recipient.id = cn.recipient_id
WHERE
    character_id = ?
ORDER BY
    timestamp DESC;

-- name: ListCharacterNotificationsUnread :many
SELECT
    sqlc.embed(cn),
    sqlc.embed(sender),
    sqlc.embed(nt),
    recipient.name as recipient_name,
    recipient.category as recipient_category
FROM
    character_notifications cn
    JOIN eve_entities sender ON sender.id = cn.sender_id
    JOIN notification_types nt ON nt.id = cn.type_id
    LEFT JOIN eve_entities recipient ON recipient.id = cn.recipient_id
WHERE
    character_id = ?
    AND cn.is_read IS FALSE
ORDER BY
    timestamp DESC;

-- name: ListCharacterNotificationsUnprocessed :many
SELECT
    sqlc.embed(cn),
    sqlc.embed(sender),
    sqlc.embed(nt),
    recipient.name as recipient_name,
    recipient.category as recipient_category
FROM
    character_notifications cn
    JOIN eve_entities sender ON sender.id = cn.sender_id
    JOIN notification_types nt ON nt.id = cn.type_id
    LEFT JOIN eve_entities recipient ON recipient.id = cn.recipient_id
WHERE
    cn.character_id = ?
    AND cn.is_processed IS FALSE
    AND cn.title IS NOT NULL
    AND cn.body IS NOT NULL
    AND cn.timestamp > ?
    AND notification_id NOT IN (
        SELECT
            cn2.notification_id
        FROM
            character_notifications cn2
        WHERE
            cn2.is_processed IS TRUE
            AND cn2.timestamp > ?
    )
ORDER BY
    timestamp;

-- name: UpdateCharacterNotification :exec
UPDATE character_notifications
SET
    body = ?2,
    is_read = ?3,
    title = ?4
WHERE
    id = ?1;

-- name: UpdateCharacterNotificationsSetProcessed :exec
UPDATE character_notifications
SET
    is_processed = TRUE
WHERE
    notification_id = ?;

-- name: CreateNotificationType :one
INSERT INTO
    notification_types (name)
VALUES
    (?)
RETURNING
    id;

-- name: GetNotificationTypeID :one
SELECT
    id
FROM
    notification_types
WHERE
    name = ?;