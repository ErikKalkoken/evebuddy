-- name: CreateCharacterTag :one
INSERT INTO
    character_tags (name)
VALUES
    (?)
RETURNING
    *;

-- name: GetCharacterTag :one
SELECT
    *
FROM
    character_tags
WHERE
    id = ?;

-- name: DeleteCharacterTag :exec
DELETE FROM character_tags
WHERE
    id = ?;

-- name: DeleteAllCharacterTags :exec
DELETE FROM character_tags;

-- name: ListCharacterTags :many
SELECT
    *
FROM
    character_tags
ORDER BY
    name;

-- name: UpdateCharacterTagName :exec
UPDATE character_tags
SET
    name = ?
WHERE
    id = ?;

-- name: CreateCharactersCharacterTag :exec
INSERT INTO
    characters_character_tags (character_id, tag_id)
VALUES
    (?, ?);

-- name: DeleteCharactersCharacterTag :exec
DELETE FROM characters_character_tags
WHERE
    character_id = ?
    AND tag_id = ?;

-- name: ListCharactersForCharacterTag :many
SELECT
    ec.id,
    ec.name
FROM
    characters_character_tags ct
    JOIN characters c ON c.id = ct.character_id
    JOIN eve_characters ec ON ec.id = c.id
WHERE
    ct.tag_id = ?
ORDER BY
    ec.name;

-- name: ListCharacterTagsForCharacter :many
SELECT
    t.*
FROM
    characters_character_tags ct
    JOIN character_tags t ON t.id = ct.tag_id
WHERE
    ct.character_id = ?
ORDER BY
    t.name;
