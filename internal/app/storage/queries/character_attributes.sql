-- name: UpdateOrCreateCharacterAttributes :exec
INSERT INTO
    character_attributes (
        character_id,
        bonus_remaps,
        charisma,
        intelligence,
        last_remap_date,
        memory,
        perception,
        willpower
    )
VALUES
    (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)
ON CONFLICT (character_id) DO UPDATE
SET
    bonus_remaps = ?2,
    charisma = ?3,
    intelligence = ?4,
    last_remap_date = ?5,
    memory = ?6,
    perception = ?7,
    willpower = ?8
WHERE
    character_id = ?1;

-- name: GetCharacterAttributes :one
SELECT
    *
FROM
    character_attributes
WHERE
    character_id = ?1;
