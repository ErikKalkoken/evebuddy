// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: character_mails.sql

package queries

import (
	"context"
	"time"
)

const createMail = `-- name: CreateMail :one
INSERT INTO
    character_mails (
        body,
        character_id,
        from_id,
        is_processed,
        is_read,
        mail_id,
        subject,
        timestamp
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id, body, character_id, from_id, is_processed, is_read, mail_id, subject, timestamp
`

type CreateMailParams struct {
	Body        string
	CharacterID int64
	FromID      int64
	IsProcessed bool
	IsRead      bool
	MailID      int64
	Subject     string
	Timestamp   time.Time
}

func (q *Queries) CreateMail(ctx context.Context, arg CreateMailParams) (CharacterMail, error) {
	row := q.db.QueryRowContext(ctx, createMail,
		arg.Body,
		arg.CharacterID,
		arg.FromID,
		arg.IsProcessed,
		arg.IsRead,
		arg.MailID,
		arg.Subject,
		arg.Timestamp,
	)
	var i CharacterMail
	err := row.Scan(
		&i.ID,
		&i.Body,
		&i.CharacterID,
		&i.FromID,
		&i.IsProcessed,
		&i.IsRead,
		&i.MailID,
		&i.Subject,
		&i.Timestamp,
	)
	return i, err
}

const createMailCharacterMailLabel = `-- name: CreateMailCharacterMailLabel :exec
INSERT INTO
    character_mail_mail_labels (
        character_mail_label_id,
        character_mail_id
    )
VALUES
    (?, ?)
`

type CreateMailCharacterMailLabelParams struct {
	CharacterMailLabelID int64
	CharacterMailID      int64
}

func (q *Queries) CreateMailCharacterMailLabel(ctx context.Context, arg CreateMailCharacterMailLabelParams) error {
	_, err := q.db.ExecContext(ctx, createMailCharacterMailLabel, arg.CharacterMailLabelID, arg.CharacterMailID)
	return err
}

const createMailRecipient = `-- name: CreateMailRecipient :exec
INSERT INTO
    character_mails_recipients (mail_id, eve_entity_id)
VALUES
    (?, ?)
`

type CreateMailRecipientParams struct {
	MailID      int64
	EveEntityID int64
}

func (q *Queries) CreateMailRecipient(ctx context.Context, arg CreateMailRecipientParams) error {
	_, err := q.db.ExecContext(ctx, createMailRecipient, arg.MailID, arg.EveEntityID)
	return err
}

const deleteMail = `-- name: DeleteMail :exec
DELETE FROM
    character_mails
WHERE
    character_mails.character_id = ?
    AND character_mails.mail_id = ?
`

type DeleteMailParams struct {
	CharacterID int64
	MailID      int64
}

func (q *Queries) DeleteMail(ctx context.Context, arg DeleteMailParams) error {
	_, err := q.db.ExecContext(ctx, deleteMail, arg.CharacterID, arg.MailID)
	return err
}

const deleteMailCharacterMailLabels = `-- name: DeleteMailCharacterMailLabels :exec
DELETE FROM
    character_mail_mail_labels
WHERE
    character_mail_mail_labels.character_mail_id = ?
`

func (q *Queries) DeleteMailCharacterMailLabels(ctx context.Context, characterMailID int64) error {
	_, err := q.db.ExecContext(ctx, deleteMailCharacterMailLabels, characterMailID)
	return err
}

const getAllMailUnreadCount = `-- name: GetAllMailUnreadCount :one
SELECT
    COUNT(*)
FROM
    character_mails
WHERE
    is_read IS FALSE
`

func (q *Queries) GetAllMailUnreadCount(ctx context.Context) (int64, error) {
	row := q.db.QueryRowContext(ctx, getAllMailUnreadCount)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const getCharacterMailLabelUnreadCounts = `-- name: GetCharacterMailLabelUnreadCounts :many
SELECT
    label_id,
    COUNT(cm.id) AS unread_count_2
FROM
    character_mail_labels cml
    JOIN character_mail_mail_labels cmml ON cmml.character_mail_label_id = cml.id
    JOIN character_mails cm ON cm.id = cmml.character_mail_id
WHERE
    cml.character_id = ?
    AND is_read IS FALSE
GROUP BY
    label_id
`

type GetCharacterMailLabelUnreadCountsRow struct {
	LabelID      int64
	UnreadCount2 int64
}

func (q *Queries) GetCharacterMailLabelUnreadCounts(ctx context.Context, characterID int64) ([]GetCharacterMailLabelUnreadCountsRow, error) {
	rows, err := q.db.QueryContext(ctx, getCharacterMailLabelUnreadCounts, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetCharacterMailLabelUnreadCountsRow
	for rows.Next() {
		var i GetCharacterMailLabelUnreadCountsRow
		if err := rows.Scan(&i.LabelID, &i.UnreadCount2); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getCharacterMailLabels = `-- name: GetCharacterMailLabels :many
SELECT
    character_mail_labels.id, character_mail_labels.character_id, character_mail_labels.color, character_mail_labels.label_id, character_mail_labels.name, character_mail_labels.unread_count
FROM
    character_mail_labels
    JOIN character_mail_mail_labels ON character_mail_mail_labels.character_mail_label_id = character_mail_labels.id
WHERE
    character_mail_id = ?
`

func (q *Queries) GetCharacterMailLabels(ctx context.Context, characterMailID int64) ([]CharacterMailLabel, error) {
	rows, err := q.db.QueryContext(ctx, getCharacterMailLabels, characterMailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []CharacterMailLabel
	for rows.Next() {
		var i CharacterMailLabel
		if err := rows.Scan(
			&i.ID,
			&i.CharacterID,
			&i.Color,
			&i.LabelID,
			&i.Name,
			&i.UnreadCount,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getCharacterMailListUnreadCounts = `-- name: GetCharacterMailListUnreadCounts :many
SELECT
    eve_entities.id AS list_id,
    COUNT(cm.id) as unread_count_2
FROM
    character_mails cm
    JOIN character_mails_recipients ON character_mails_recipients.mail_id = cm.id
    JOIN eve_entities ON eve_entities.id = character_mails_recipients.eve_entity_id
WHERE
    character_id = ?
    AND eve_entities.category = "mail_list"
    AND cm.is_read IS FALSE
GROUP BY
    eve_entities.id
`

type GetCharacterMailListUnreadCountsRow struct {
	ListID       int64
	UnreadCount2 int64
}

func (q *Queries) GetCharacterMailListUnreadCounts(ctx context.Context, characterID int64) ([]GetCharacterMailListUnreadCountsRow, error) {
	rows, err := q.db.QueryContext(ctx, getCharacterMailListUnreadCounts, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetCharacterMailListUnreadCountsRow
	for rows.Next() {
		var i GetCharacterMailListUnreadCountsRow
		if err := rows.Scan(&i.ListID, &i.UnreadCount2); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getMail = `-- name: GetMail :one
SELECT
    cm.id, cm.body, cm.character_id, cm.from_id, cm.is_processed, cm.is_read, cm.mail_id, cm.subject, cm.timestamp,
    ee.id, ee.category, ee.name
FROM
    character_mails cm
    JOIN eve_entities ee ON ee.id = cm.from_id
WHERE
    character_id = ?
    AND mail_id = ?
`

type GetMailParams struct {
	CharacterID int64
	MailID      int64
}

type GetMailRow struct {
	CharacterMail CharacterMail
	EveEntity     EveEntity
}

func (q *Queries) GetMail(ctx context.Context, arg GetMailParams) (GetMailRow, error) {
	row := q.db.QueryRowContext(ctx, getMail, arg.CharacterID, arg.MailID)
	var i GetMailRow
	err := row.Scan(
		&i.CharacterMail.ID,
		&i.CharacterMail.Body,
		&i.CharacterMail.CharacterID,
		&i.CharacterMail.FromID,
		&i.CharacterMail.IsProcessed,
		&i.CharacterMail.IsRead,
		&i.CharacterMail.MailID,
		&i.CharacterMail.Subject,
		&i.CharacterMail.Timestamp,
		&i.EveEntity.ID,
		&i.EveEntity.Category,
		&i.EveEntity.Name,
	)
	return i, err
}

const getMailCount = `-- name: GetMailCount :one
SELECT
    COUNT(*)
FROM
    character_mails
WHERE
    character_mails.character_id = ?
`

func (q *Queries) GetMailCount(ctx context.Context, characterID int64) (int64, error) {
	row := q.db.QueryRowContext(ctx, getMailCount, characterID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const getMailRecipients = `-- name: GetMailRecipients :many
SELECT
    eve_entities.id, eve_entities.category, eve_entities.name
FROM
    eve_entities
    JOIN character_mails_recipients ON character_mails_recipients.eve_entity_id = eve_entities.id
WHERE
    mail_id = ?
`

func (q *Queries) GetMailRecipients(ctx context.Context, mailID int64) ([]EveEntity, error) {
	rows, err := q.db.QueryContext(ctx, getMailRecipients, mailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []EveEntity
	for rows.Next() {
		var i EveEntity
		if err := rows.Scan(&i.ID, &i.Category, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getMailUnreadCount = `-- name: GetMailUnreadCount :one
SELECT
    COUNT(*)
FROM
    character_mails
WHERE
    character_mails.character_id = ?
    AND is_read IS FALSE
`

func (q *Queries) GetMailUnreadCount(ctx context.Context, characterID int64) (int64, error) {
	row := q.db.QueryRowContext(ctx, getMailUnreadCount, characterID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const listMailIDs = `-- name: ListMailIDs :many
SELECT
    mail_id
FROM
    character_mails
WHERE
    character_id = ?
`

func (q *Queries) ListMailIDs(ctx context.Context, characterID int64) ([]int64, error) {
	rows, err := q.db.QueryContext(ctx, listMailIDs, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int64
	for rows.Next() {
		var mail_id int64
		if err := rows.Scan(&mail_id); err != nil {
			return nil, err
		}
		items = append(items, mail_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMailsForLabelOrdered = `-- name: ListMailsForLabelOrdered :many
SELECT
    cm.id, cm.body, cm.character_id, cm.from_id, cm.is_processed, cm.is_read, cm.mail_id, cm.subject, cm.timestamp,
    ee.id, ee.category, ee.name
FROM
    character_mails cm
    JOIN eve_entities ee ON ee.id = cm.from_id
    JOIN character_mail_mail_labels cml ON cml.character_mail_id = cm.id
    JOIN character_mail_labels ON character_mail_labels.id = cml.character_mail_label_id
WHERE
    cm.character_id = ?
    AND label_id = ?
ORDER BY
    timestamp DESC
`

type ListMailsForLabelOrderedParams struct {
	CharacterID int64
	LabelID     int64
}

type ListMailsForLabelOrderedRow struct {
	CharacterMail CharacterMail
	EveEntity     EveEntity
}

func (q *Queries) ListMailsForLabelOrdered(ctx context.Context, arg ListMailsForLabelOrderedParams) ([]ListMailsForLabelOrderedRow, error) {
	rows, err := q.db.QueryContext(ctx, listMailsForLabelOrdered, arg.CharacterID, arg.LabelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListMailsForLabelOrderedRow
	for rows.Next() {
		var i ListMailsForLabelOrderedRow
		if err := rows.Scan(
			&i.CharacterMail.ID,
			&i.CharacterMail.Body,
			&i.CharacterMail.CharacterID,
			&i.CharacterMail.FromID,
			&i.CharacterMail.IsProcessed,
			&i.CharacterMail.IsRead,
			&i.CharacterMail.MailID,
			&i.CharacterMail.Subject,
			&i.CharacterMail.Timestamp,
			&i.EveEntity.ID,
			&i.EveEntity.Category,
			&i.EveEntity.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMailsForListOrdered = `-- name: ListMailsForListOrdered :many
SELECT
    cm.id, cm.body, cm.character_id, cm.from_id, cm.is_processed, cm.is_read, cm.mail_id, cm.subject, cm.timestamp,
    ee.id, ee.category, ee.name
FROM
    character_mails cm
    JOIN eve_entities ee ON ee.id = cm.from_id
    JOIN character_mails_recipients cmr ON cmr.mail_id = cm.id
WHERE
    character_id = ?
    AND cmr.eve_entity_id = ?
ORDER BY
    timestamp DESC
`

type ListMailsForListOrderedParams struct {
	CharacterID int64
	EveEntityID int64
}

type ListMailsForListOrderedRow struct {
	CharacterMail CharacterMail
	EveEntity     EveEntity
}

func (q *Queries) ListMailsForListOrdered(ctx context.Context, arg ListMailsForListOrderedParams) ([]ListMailsForListOrderedRow, error) {
	rows, err := q.db.QueryContext(ctx, listMailsForListOrdered, arg.CharacterID, arg.EveEntityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListMailsForListOrderedRow
	for rows.Next() {
		var i ListMailsForListOrderedRow
		if err := rows.Scan(
			&i.CharacterMail.ID,
			&i.CharacterMail.Body,
			&i.CharacterMail.CharacterID,
			&i.CharacterMail.FromID,
			&i.CharacterMail.IsProcessed,
			&i.CharacterMail.IsRead,
			&i.CharacterMail.MailID,
			&i.CharacterMail.Subject,
			&i.CharacterMail.Timestamp,
			&i.EveEntity.ID,
			&i.EveEntity.Category,
			&i.EveEntity.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMailsNoLabelOrdered = `-- name: ListMailsNoLabelOrdered :many
SELECT
    cm.id, cm.body, cm.character_id, cm.from_id, cm.is_processed, cm.is_read, cm.mail_id, cm.subject, cm.timestamp,
    ee.id, ee.category, ee.name
FROM
    character_mails cm
    JOIN eve_entities ee ON ee.id = cm.from_id
    LEFT JOIN character_mail_mail_labels cml ON cml.character_mail_id = cm.id
WHERE
    character_id = ?
    AND cml.character_mail_id IS NULL
ORDER BY
    timestamp DESC
`

type ListMailsNoLabelOrderedRow struct {
	CharacterMail CharacterMail
	EveEntity     EveEntity
}

func (q *Queries) ListMailsNoLabelOrdered(ctx context.Context, characterID int64) ([]ListMailsNoLabelOrderedRow, error) {
	rows, err := q.db.QueryContext(ctx, listMailsNoLabelOrdered, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListMailsNoLabelOrderedRow
	for rows.Next() {
		var i ListMailsNoLabelOrderedRow
		if err := rows.Scan(
			&i.CharacterMail.ID,
			&i.CharacterMail.Body,
			&i.CharacterMail.CharacterID,
			&i.CharacterMail.FromID,
			&i.CharacterMail.IsProcessed,
			&i.CharacterMail.IsRead,
			&i.CharacterMail.MailID,
			&i.CharacterMail.Subject,
			&i.CharacterMail.Timestamp,
			&i.EveEntity.ID,
			&i.EveEntity.Category,
			&i.EveEntity.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMailsOrdered = `-- name: ListMailsOrdered :many
SELECT
    cm.id, cm.body, cm.character_id, cm.from_id, cm.is_processed, cm.is_read, cm.mail_id, cm.subject, cm.timestamp,
    ee.id, ee.category, ee.name
FROM
    character_mails cm
    JOIN eve_entities ee ON ee.id = cm.from_id
WHERE
    character_id = ?
ORDER BY
    timestamp DESC
`

type ListMailsOrderedRow struct {
	CharacterMail CharacterMail
	EveEntity     EveEntity
}

func (q *Queries) ListMailsOrdered(ctx context.Context, characterID int64) ([]ListMailsOrderedRow, error) {
	rows, err := q.db.QueryContext(ctx, listMailsOrdered, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListMailsOrderedRow
	for rows.Next() {
		var i ListMailsOrderedRow
		if err := rows.Scan(
			&i.CharacterMail.ID,
			&i.CharacterMail.Body,
			&i.CharacterMail.CharacterID,
			&i.CharacterMail.FromID,
			&i.CharacterMail.IsProcessed,
			&i.CharacterMail.IsRead,
			&i.CharacterMail.MailID,
			&i.CharacterMail.Subject,
			&i.CharacterMail.Timestamp,
			&i.EveEntity.ID,
			&i.EveEntity.Category,
			&i.EveEntity.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMailsUnprocessed = `-- name: ListMailsUnprocessed :many
SELECT
    cm.id, cm.body, cm.character_id, cm.from_id, cm.is_processed, cm.is_read, cm.mail_id, cm.subject, cm.timestamp,
    ee.id, ee.category, ee.name
FROM
    character_mails cm
    JOIN eve_entities ee ON ee.id = cm.from_id
    LEFT JOIN character_mail_mail_labels cml ON cml.character_mail_id = cm.id
    LEFT JOIN character_mail_labels ON character_mail_labels.id = cml.character_mail_label_id
WHERE
    cm.character_id = ?
    AND (
        label_id <> ?
        OR label_id IS NULL
    )
    AND is_processed = FALSE
    AND timestamp > ?
ORDER BY
    timestamp ASC
`

type ListMailsUnprocessedParams struct {
	CharacterID int64
	LabelID     int64
	Timestamp   time.Time
}

type ListMailsUnprocessedRow struct {
	CharacterMail CharacterMail
	EveEntity     EveEntity
}

func (q *Queries) ListMailsUnprocessed(ctx context.Context, arg ListMailsUnprocessedParams) ([]ListMailsUnprocessedRow, error) {
	rows, err := q.db.QueryContext(ctx, listMailsUnprocessed, arg.CharacterID, arg.LabelID, arg.Timestamp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListMailsUnprocessedRow
	for rows.Next() {
		var i ListMailsUnprocessedRow
		if err := rows.Scan(
			&i.CharacterMail.ID,
			&i.CharacterMail.Body,
			&i.CharacterMail.CharacterID,
			&i.CharacterMail.FromID,
			&i.CharacterMail.IsProcessed,
			&i.CharacterMail.IsRead,
			&i.CharacterMail.MailID,
			&i.CharacterMail.Subject,
			&i.CharacterMail.Timestamp,
			&i.EveEntity.ID,
			&i.EveEntity.Category,
			&i.EveEntity.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listMailsUnreadOrdered = `-- name: ListMailsUnreadOrdered :many
SELECT
    cm.id, cm.body, cm.character_id, cm.from_id, cm.is_processed, cm.is_read, cm.mail_id, cm.subject, cm.timestamp,
    ee.id, ee.category, ee.name
FROM
    character_mails cm
    JOIN eve_entities ee ON ee.id = cm.from_id
WHERE
    character_id = ?
    AND is_read IS FALSE
ORDER BY
    timestamp DESC
`

type ListMailsUnreadOrderedRow struct {
	CharacterMail CharacterMail
	EveEntity     EveEntity
}

func (q *Queries) ListMailsUnreadOrdered(ctx context.Context, characterID int64) ([]ListMailsUnreadOrderedRow, error) {
	rows, err := q.db.QueryContext(ctx, listMailsUnreadOrdered, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListMailsUnreadOrderedRow
	for rows.Next() {
		var i ListMailsUnreadOrderedRow
		if err := rows.Scan(
			&i.CharacterMail.ID,
			&i.CharacterMail.Body,
			&i.CharacterMail.CharacterID,
			&i.CharacterMail.FromID,
			&i.CharacterMail.IsProcessed,
			&i.CharacterMail.IsRead,
			&i.CharacterMail.MailID,
			&i.CharacterMail.Subject,
			&i.CharacterMail.Timestamp,
			&i.EveEntity.ID,
			&i.EveEntity.Category,
			&i.EveEntity.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateCharacterMailIsRead = `-- name: UpdateCharacterMailIsRead :exec
UPDATE
    character_mails
SET
    is_read = ?2
WHERE
    id = ?1
`

type UpdateCharacterMailIsReadParams struct {
	ID     int64
	IsRead bool
}

func (q *Queries) UpdateCharacterMailIsRead(ctx context.Context, arg UpdateCharacterMailIsReadParams) error {
	_, err := q.db.ExecContext(ctx, updateCharacterMailIsRead, arg.ID, arg.IsRead)
	return err
}

const updateCharacterMailSetProcessed = `-- name: UpdateCharacterMailSetProcessed :exec
UPDATE
    character_mails
SET
    is_processed = TRUE
WHERE
    id = ?1
`

func (q *Queries) UpdateCharacterMailSetProcessed(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, updateCharacterMailSetProcessed, id)
	return err
}
