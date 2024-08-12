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

-- name: ListCharacterNotificationsTypes :many
SELECT sqlc.embed(cn), sqlc.embed(ee), sqlc.embed(nt)
FROM character_notifications cn
JOIN eve_entities ee ON ee.id = cn.sender_id
JOIN notification_types nt ON nt.id = cn.type_id
WHERE character_id = ?
AND nt.name IN (sqlc.slice('names'))
ORDER BY timestamp DESC;

-- name: ListCharacterNotificationsUnread :many
SELECT sqlc.embed(cn), sqlc.embed(ee), sqlc.embed(nt)
FROM character_notifications cn
JOIN eve_entities ee ON ee.id = cn.sender_id
JOIN notification_types nt ON nt.id = cn.type_id
WHERE character_id = ?
AND cn.is_read IS FALSE
ORDER BY timestamp DESC;

-- name: CalcCharacterNotificationUnreadCounts :many
SELECT cn.type_id, nt.name, SUM(NOT cn.is_read)
FROM character_notifications cn
JOIN notification_types nt ON nt.id = cn.type_id
WHERE character_id = ?
GROUP BY cn.type_id, nt.name;

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
