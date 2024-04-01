package model

import (
	"database/sql"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

var bodyPolicy = bluemonday.StrictPolicy()

// An Eve mail belonging to a character
type Mail struct {
	Body        string
	CharacterID int32 `db:"character_id"`
	Character   Character
	FromID      int32     `db:"from_id"`
	From        EveEntity `db:"eve_entity"`
	Labels      []MailLabel
	IsRead      bool `db:"is_read"`
	ID          uint64
	MailID      int32 `db:"mail_id"`
	Recipients  []EveEntity
	Subject     string
	Timestamp   time.Time
}

// Create creates a new mail
func (m *Mail) Create() error {
	if m.Character.ID != 0 {
		m.CharacterID = m.Character.ID
	}
	if m.CharacterID == 0 {
		return fmt.Errorf("CharacterID can not be zero")
	}
	if m.From.ID != 0 {
		m.FromID = m.From.ID
	}
	if m.FromID == 0 {
		return fmt.Errorf("FormID can not be zero")
	}
	r, err := db.NamedExec(`
		INSERT INTO mails (
			body,
			character_id,
			from_id,
			is_read,
			mail_id,
			subject,
			timestamp
		)
		VALUES (
			:body,
			:character_id,
			:from_id,
			:is_read,
			:mail_id,
			:subject,
			:timestamp
		)`,
		*m,
	)
	if err != nil {
		return err
	}
	newID, err := r.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = uint64(newID)
	if err := m.addRecipients(); err != nil {
		return err
	}
	if err := m.addLabels(); err != nil {
		return err
	}
	return nil
}

func (m *Mail) addRecipients() error {
	for _, r := range m.Recipients {
		_, err := db.Exec(`
			INSERT INTO mail_recipients (mail_id, eve_entity_id)
			VALUES (?, ?);
			`,
			m.ID,
			r.ID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Mail) addLabels() error {
	for _, l := range m.Labels {
		_, err := db.Exec(`
			INSERT INTO mail_mail_labels (mail_label_id, mail_id)
			VALUES (?, ?);
			`,
			l.ID,
			m.ID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// BodyPlain returns a mail's body as plain text.
func (m *Mail) BodyPlain() string {
	t := strings.ReplaceAll(m.Body, "<br>", "\n")
	b := html.UnescapeString(bodyPolicy.Sanitize(t))
	return b
}

// BodyForward returns a mail's body for a mail forward or reply
func (m *Mail) ToString(format string) string {
	s := "\n---\n"
	s += m.MakeHeaderText(format)
	s += "\n\n"
	s += m.BodyPlain()
	return s
}

func (m *Mail) MakeHeaderText(format string) string {
	var names []string
	for _, n := range m.Recipients {
		names = append(names, n.Name)
	}
	header := fmt.Sprintf(
		"From: %s\nSent: %s\nTo: %s",
		m.From.Name,
		m.Timestamp.Format(format),
		strings.Join(names, ", "),
	)
	return header
}

// FetchMail returns a mail.
func FetchMail(id uint64) (*Mail, error) {
	row := db.QueryRow(
		`SELECT *
		FROM mails
		JOIN eve_entities ON eve_entities.id = mails.from_id
		WHERE mails.id = ?;`,
		id,
	)
	var m Mail
	err := row.Scan(
		&m.ID,
		&m.Body,
		&m.CharacterID,
		&m.FromID,
		&m.IsRead,
		&m.MailID,
		&m.Subject,
		&m.Timestamp,
		&m.From.ID,
		&m.From.Category,
		&m.From.Name,
	)
	if err != nil {
		return nil, err
	}
	m.Recipients, err = fetchMailRecipients(m.ID)
	if err != nil {
		return nil, err
	}
	m.Labels, err = fetchMailLabels(m.ID)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func fetchMailRecipients(mailID uint64) ([]EveEntity, error) {
	var rr []EveEntity
	err := db.Select(
		&rr,
		`SELECT eve_entities.*
		FROM eve_entities
		JOIN mail_recipients ON mail_recipients.eve_entity_id = eve_entities.id
		WHERE mail_id = ?
		`, mailID,
	)
	return rr, err
}

func fetchMailLabels(mailID uint64) ([]MailLabel, error) {
	var ll []MailLabel
	err := db.Select(
		&ll,
		`SELECT mail_labels.*
		FROM mail_labels
		JOIN mail_mail_labels ON mail_mail_labels.mail_label_id = mail_labels.id
		WHERE mail_id = ?
		`, mailID,
	)
	return ll, err
}

// FetchMailIDs return mail IDs of all existing mails for a character
func FetchMailIDs(characterID int32) ([]int32, error) {
	var ids []int32
	err := db.Select(&ids, "SELECT mail_id FROM mails WHERE character_id = ?", characterID)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FetchMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func FetchMailsForLabel(characterID int32, labelID int32) ([]Mail, error) {
	var rows *sql.Rows
	switch labelID {
	case LabelAll:
		sql := `
			SELECT mails.*, eve_entities.*
			FROM mails
			JOIN eve_entities ON eve_entities.id = mails.from_id
			WHERE character_id = ?
			ORDER BY timestamp DESC
		`
		r, err := db.Query(sql, characterID)
		if err != nil {
			return nil, err
		}
		rows = r
	case LabelNone:
		sql := `
			SELECT mails.*, eve_entities.*
			FROM mails
			LEFT JOIN mail_mail_labels ON mail_mail_labels.mail_id = mails.id
			JOIN eve_entities ON eve_entities.id = mails.from_id
			WHERE character_id = ?
			AND mail_mail_labels.mail_id IS NULL
			ORDER BY timestamp DESC
		`
		r, err := db.Query(sql, characterID)
		if err != nil {
			return nil, err
		}
		rows = r
	default:
		sql := `
			SELECT mails.*, eve_entities.*
			FROM mails
			JOIN mail_mail_labels ON mail_mail_labels.mail_id = mails.id
			JOIN mail_labels ON mail_labels.id = mail_mail_labels.mail_label_id
			JOIN eve_entities ON eve_entities.id = mails.from_id
			WHERE mails.character_id = ?
			AND label_id = ?
			ORDER BY timestamp DESC
		`
		r, err := db.Query(sql, characterID, labelID)
		if err != nil {
			return nil, err
		}
		rows = r
	}
	mm, err := scanMail(rows)
	slog.Debug("Fetched mails for label", "labelID", labelID, "count", len(mm))
	return mm, err
}

func FetchMailsForList(characterID int32, listID int32) ([]Mail, error) {
	sql := `
		SELECT mails.*, eve_entities.*
		FROM mails
		JOIN eve_entities ON eve_entities.id = mails.from_id
		JOIN mail_recipients ON mail_recipients.mail_id = mails.id
		WHERE character_id = ?
		AND mail_recipients.eve_entity_id = ?
		ORDER BY timestamp DESC
	`
	rows, err := db.Query(sql, characterID, listID)
	if err != nil {
		return nil, err
	}
	mm, err := scanMail(rows)
	slog.Debug("Fetched mails for list", "listID", listID, "count", len(mm))
	return mm, err
}

func scanMail(rows *sql.Rows) ([]Mail, error) {
	var mm []Mail
	for rows.Next() {
		var m Mail
		err := rows.Scan(
			&m.ID,
			&m.Body,
			&m.CharacterID,
			&m.FromID,
			&m.IsRead,
			&m.MailID,
			&m.Subject,
			&m.Timestamp,
			&m.From.ID,
			&m.From.Category,
			&m.From.Name,
		)
		if err != nil {
			return nil, err
		}
		mm = append(mm, m)
	}
	return mm, nil
}

func Test() {

	// var l MailLabel
	// db.First(&l, 4)

	// var mm []Mail
	// err := db.Model(&l).Association("Mails").Find(&mm)
	// slog.Info("result", "mails", mm, "mailsCount", len(mm), "error", err)

	panic("Stop")
}
