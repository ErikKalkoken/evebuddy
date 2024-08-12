-- name: CreateCharacterNotification :exec
INSERT INTO character_notifications (
    character_id,
    is_read,
    notification_id,
    sender_id,
    text,
    timestamp,
    type_id
)
VALUES (
    ?, ?, ?, ?, ?, ?, ?
);

-- name: GetCharacterNotification :one
SELECT sqlc.embed(cn), sqlc.embed(ee), sqlc.embed(nt)
FROM character_notifications cn
JOIN eve_entities ee ON ee.id = cn.sender_id
JOIN notification_types nt ON nt.id = cn.type_id
WHERE character_id = ? and notification_id = ?;

-- name: ListCharacterNotificationIDs :many
SELECT notification_id
FROM character_notifications
WHERE character_id = ?;

-- name: ListCharacterNotifications :many
SELECT sqlc.embed(cn), sqlc.embed(ee), sqlc.embed(nt)
FROM character_notifications cn
JOIN eve_entities ee ON ee.id = cn.sender_id
JOIN notification_types nt ON nt.id = cn.type_id
WHERE character_id = ?
ORDER BY timestamp DESC;

-- name: UpdateCharacterNotificationsIsRead :exec
UPDATE character_notifications
SET is_read = ?2
WHERE id = ?1;

-- name: CreateNotificationType :one
INSERT INTO notification_types (
    name
)
VALUES (
    ?
)
RETURNING id;

-- name: GetNotificationTypeID :one
SELECT id
FROM notification_types
WHERE name = ?;
