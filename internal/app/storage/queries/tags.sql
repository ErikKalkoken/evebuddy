-- name: CreateTag :one
INSERT INTO
    tags (name, description)
VALUES
    (?, ?) RETURNING *;

-- name: GetTag :one
SELECT
    *
FROM
    tags
WHERE
    id = ?;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE
    id = ?;

-- name: ListTagsByName :many
SELECT
    *
FROM
    tags
ORDER BY
    name;

-- name: UpdateTagName :exec
UPDATE tags
SET
    name = ?
WHERE
    id = ?;

-- name: UpdateTagDescription :exec
UPDATE tags
SET
    description = ?
WHERE
    id = ?;


-- name: CreateCharacterTag :exec
INSERT INTO
    characters_tags (character_id, tag_id)
VALUES
    (?, ?);

-- name: DeleteCharacterTag :exec
DELETE FROM characters_tags
WHERE
    character_id = ?
    AND tag_id = ?;

-- name: ListCharactersForTag :many
SELECT
    ec.id, ec.name
FROM
    characters_tags ct
    JOIN characters c ON c.id = ct.character_id
    JOIN eve_characters ec ON ec.id = c.id
WHERE
    ct.tag_id = ?
ORDER BY
    ec.name;

-- name: ListTagsForCharacter :many
SELECT
    t.*
FROM
    characters_tags ct
    JOIN tags t ON t.id = ct.tag_id
WHERE
    ct.character_id = ?
ORDER BY
    t.name;
