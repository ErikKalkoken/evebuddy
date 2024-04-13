package logic

import (
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"fmt"
	"html"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/antihax/goesi/esi"
	"github.com/antihax/goesi/optional"
	"github.com/microcosm-cc/bluemonday"
)

const (
	maxMails          = 1000
	maxHeadersPerPage = 50 // maximum header objects returned per page
)

var bodyPolicy = bluemonday.StrictPolicy()

// An Eve mail belonging to a character
type Mail struct {
	Body        string
	CharacterID int32
	From        EveEntity
	Labels      []MailLabel
	IsRead      bool
	ID          uint64
	MailID      int32
	Recipients  []EveEntity
	Subject     string
	Timestamp   time.Time
}

func mailFromDBModel(m model.Mail) Mail {
	if m.CharacterID == 0 {
		panic("missing character ID")
	}
	m2 := Mail{
		Body:        m.Body,
		CharacterID: m.CharacterID,
		From:        eveEntityFromDBModel(m.From),
		IsRead:      m.IsRead,
		ID:          m.ID,
		MailID:      m.MailID,
		Subject:     m.Subject,
		Timestamp:   m.Timestamp,
	}
	for _, r := range m.Recipients {
		m2.Recipients = append(m2.Recipients, eveEntityFromDBModel(r))
	}
	return m2
}

// Save updates to an existing mail.
func (m *Mail) Save() error {
	m2 := model.Mail{
		Body:        m.Body,
		CharacterID: m.CharacterID,
		FromID:      m.From.ID,
		IsRead:      m.IsRead,
		ID:          m.ID,
		MailID:      m.MailID,
		Subject:     m.Subject,
		Timestamp:   m.Timestamp,
	}
	err := m2.Save()
	if err != nil {
		return err
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

// DeleteMail deletes a mail both on ESI and in the database.
func (m *Mail) Delete() error {
	token, err := GetValidToken(m.CharacterID)
	if err != nil {
		return err
	}
	_, err = esiClient.ESI.MailApi.DeleteCharactersCharacterIdMailMailId(token.NewContext(), m.CharacterID, m.MailID, nil)
	if err != nil {
		return err
	}
	_, err = model.DeleteMail(m.ID)
	if err != nil {
		return err
	}
	return nil
}

// SendMail created a new mail on ESI stores it locally.
func SendMail(characterID int32, subject string, recipients []esi.PostCharactersCharacterIdMailRecipient, body string) error {
	if subject == "" {
		return fmt.Errorf("missing subject")
	}
	if body == "" {
		return fmt.Errorf("missing body")
	}
	if len(recipients) == 0 {
		return fmt.Errorf("missing recipients")
	}
	token, err := GetValidToken(characterID)
	if err != nil {
		return err
	}
	mail := esi.PostCharactersCharacterIdMailMail{
		Body:       body,
		Subject:    subject,
		Recipients: recipients,
	}
	mailID, _, err := esiClient.ESI.MailApi.PostCharactersCharacterIdMail(token.NewContext(), characterID, mail, nil)
	if err != nil {
		return err
	}
	ids := []int32{characterID}
	for _, r := range recipients {
		ids = append(ids, r.RecipientId)
	}
	_, err = addMissingEveEntities(ids)
	if err != nil {
		return err
	}
	from, err := model.GetEveEntity(token.CharacterID)
	if err != nil {
		return err
	}
	// FIXME: Ensure this still works when no labels have yet been loaded from ESI
	label, err := model.GetMailLabel(token.CharacterID, model.LabelSent)
	if err != nil {
		return err
	}
	var rr []model.EveEntity
	for _, r := range recipients {
		e, err := model.GetEveEntity(r.RecipientId)
		if err != nil {
			return err
		}
		rr = append(rr, e)
	}
	m := model.Mail{
		Body:        body,
		CharacterID: token.CharacterID,
		From:        from,
		Labels:      []model.MailLabel{label},
		MailID:      mailID,
		Recipients:  rr,
		Subject:     subject,
		IsRead:      true,
		Timestamp:   time.Now(),
	}
	if err := m.Create(); err != nil {
		return err
	}
	return nil
}

// UpdateMailRead updates an existing mail as read
func (m *Mail) UpdateMailRead() error {
	token, err := GetValidToken(m.CharacterID)
	if err != nil {
		return err
	}
	labelIDs := make([]int32, len(m.Labels))
	for i, l := range m.Labels {
		labelIDs[i] = l.LabelID
	}
	contents := esi.PutCharactersCharacterIdMailMailIdContents{Read: true, Labels: labelIDs}
	_, err = esiClient.ESI.MailApi.PutCharactersCharacterIdMailMailId(token.NewContext(), m.CharacterID, contents, m.MailID, nil)
	if err != nil {
		return err
	}
	m.IsRead = true
	if err := m.Save(); err != nil {
		return err
	}
	return nil

}

func GetMailFromDB(characterID int32, mailID int32) (Mail, error) {
	m, err := model.GetMail(characterID, mailID)
	if err != nil {
		return Mail{}, err
	}
	return mailFromDBModel(m), nil
}

// FIXME: Delete obsolete labels and mail lists
// TODO: Add ability to update existing mails for is_read and labels

// FetchMail fetches and stores new mails from ESI for a character.
func FetchMail(characterID int32, status binding.String) error {
	token, err := GetValidToken(characterID)
	if err != nil {
		return err
	}
	character, err := model.GetCharacter(characterID)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("Checking for new mail for %v", character.Name)
	status.Set(s)
	if err := updateMailLists(token); err != nil {
		return err
	}
	if err := updateMailLabels(token); err != nil {
		return err
	}
	headers, err := listMailHeaders(token)
	if err != nil {
		return err
	}
	err = updateMails(token, headers, status)
	if err != nil {
		return err
	}
	character.MailUpdatedAt.Time = time.Now()
	character.MailUpdatedAt.Valid = true
	character.Save()
	return nil
}

func updateMailLabels(token *Token) error {
	if err := token.EnsureValid(); err != nil {
		return err
	}
	ll, _, err := esiClient.ESI.MailApi.GetCharactersCharacterIdMailLabels(token.NewContext(), token.CharacterID, nil)
	if err != nil {
		return err
	}
	labels := ll.Labels
	slog.Info("Received mail labels from ESI", "count", len(labels), "characterID", token.CharacterID)
	for _, o := range labels {
		l := model.MailLabel{
			CharacterID: token.CharacterID,
			LabelID:     o.LabelId,
			Color:       o.Color,
			Name:        o.Name,
			UnreadCount: o.UnreadCount,
		}
		if err := l.Save(); err != nil {
			slog.Error("Failed to update mail label", "labelID", o.LabelId, "characterID", token.CharacterID, "error", err)
		}
	}
	return nil
}

func updateMailLists(token *Token) error {
	if err := token.EnsureValid(); err != nil {
		return err
	}
	ctx := token.NewContext()
	lists, _, err := esiClient.ESI.MailApi.GetCharactersCharacterIdMailLists(ctx, token.CharacterID, nil)
	if err != nil {
		return err
	}
	for _, o := range lists {
		e := model.EveEntity{ID: o.MailingListId, Name: o.Name, Category: model.EveEntityMailList}
		if err := e.Save(); err != nil {
			return err
		}
		m := model.MailList{CharacterID: token.CharacterID, EveEntity: e}
		if err := m.CreateIfNew(); err != nil {
			return err
		}
	}
	return nil
}

// listMailHeaders fetched mail headers from ESI with paging and returns them.
func listMailHeaders(token *Token) ([]esi.GetCharactersCharacterIdMail200Ok, error) {
	if err := token.EnsureValid(); err != nil {
		return nil, err
	}
	var mm []esi.GetCharactersCharacterIdMail200Ok
	lastMailID := int32(0)
	for {
		var opts *esi.GetCharactersCharacterIdMailOpts
		if lastMailID > 0 {
			l := optional.NewInt32(lastMailID)
			opts = &esi.GetCharactersCharacterIdMailOpts{LastMailId: l}
		} else {
			opts = nil
		}
		objs, _, err := esiClient.ESI.MailApi.GetCharactersCharacterIdMail(token.NewContext(), token.CharacterID, opts)
		if err != nil {
			return nil, err
		}
		mm = append(mm, objs...)
		isLimitExceeded := (maxMails != 0 && len(mm)+maxHeadersPerPage > maxMails)
		if len(objs) < maxHeadersPerPage || isLimitExceeded {
			break
		}
		ids := make([]int32, 0)
		for _, o := range objs {
			ids = append(ids, o.MailId)
		}
		lastMailID = slices.Min(ids)
	}
	slog.Info("Received mail headers", "characterID", token.CharacterID, "count", len(mm))
	return mm, nil
}

func updateMails(token *Token, headers []esi.GetCharactersCharacterIdMail200Ok, status binding.String) error {
	existingIDs, missingIDs, err := determineMailIDs(token.CharacterID, headers)
	if err != nil {
		return err
	}
	newMailsCount := missingIDs.Size()
	if newMailsCount == 0 {
		s := "No new mail"
		status.Set(s)
		slog.Info(s, "characterID", token.CharacterID)
		return nil
	}

	if err := token.EnsureValid(); err != nil {
		return err
	}
	character, err := model.GetCharacter(token.CharacterID)
	if err != nil {
		return err
	}

	var c atomic.Int32
	var wg sync.WaitGroup
	maxGoroutines := 20
	guard := make(chan struct{}, maxGoroutines)
	for _, header := range headers {
		if existingIDs.Has(header.MailId) {
			continue
		}
		guard <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchAndStoreMail(header, token, newMailsCount, &c, status, character.Name)
			<-guard
		}()
	}
	wg.Wait()
	total := c.Load()
	if total == 0 {
		status.Set("")
		return nil
	}
	s := fmt.Sprintf("Stored %d new mails", total)
	status.Set(s)
	slog.Info(s)
	return nil
}

func fetchAndStoreMail(header esi.GetCharactersCharacterIdMail200Ok, token *Token, newMailsCount int, c *atomic.Int32, status binding.String, characterName string) {
	entityIDs := set.New[int32]()
	entityIDs.Add(header.From)
	for _, r := range header.Recipients {
		entityIDs.Add(r.RecipientId)
	}
	_, err := addMissingEveEntities(entityIDs.ToSlice())
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	m, _, err := esiClient.ESI.MailApi.GetCharactersCharacterIdMailMailId(token.NewContext(), token.CharacterID, header.MailId, nil)
	if err != nil {
		slog.Error("Failed to process mail", "header", header, "error", err)
		return
	}
	mail := model.Mail{
		Body:        m.Body,
		CharacterID: token.CharacterID,
		MailID:      header.MailId,
		Subject:     header.Subject,
		IsRead:      header.IsRead,
	}
	mail.Timestamp = header.Timestamp
	from, err := model.GetEveEntity(header.From)
	if err != nil {
		slog.Error("Failed to parse \"from\" in mail", "header", header, "error", err)
		return
	}
	mail.From = from

	rr := fetchMailRecipients(header)
	mail.Recipients = rr

	labels, err := model.ListMailLabelsForIDs(token.CharacterID, m.Labels)
	if err != nil {
		slog.Error("Failed to resolve mail labels", "header", header, "error", err)
		return
	}
	mail.Labels = labels
	mail.Create()

	slog.Info("Created new mail", "mailID", header.MailId, "characterID", token.CharacterID)
	c.Add(1)
	current := c.Load()
	s := fmt.Sprintf("Fetched %d / %d new mails for %v", current, newMailsCount, characterName)
	status.Set(s)
}

func fetchMailRecipients(header esi.GetCharactersCharacterIdMail200Ok) []model.EveEntity {
	var rr []model.EveEntity
	for _, r := range header.Recipients {
		o, err := model.GetEveEntity(r.RecipientId)
		if err != nil {
			slog.Error("Failed to resolve mail recipient", "header", header, "recipient", r, "error", err)
			continue
		} else {
			rr = append(rr, o)
		}
	}
	return rr
}

func determineMailIDs(characterID int32, headers []esi.GetCharactersCharacterIdMail200Ok) (*set.Set[int32], *set.Set[int32], error) {
	ids, err := model.ListMailIDs(characterID)
	if err != nil {
		return nil, nil, err
	}
	existingIDs := set.NewFromSlice(ids)
	incomingIDs := set.New[int32]()
	for _, h := range headers {
		incomingIDs.Add(h.MailId)
	}
	missingIDs := incomingIDs.Difference(existingIDs)
	return existingIDs, missingIDs, nil
}

func GetMailLabelUnreadCounts(characterID int32) (map[int32]int, error) {
	return model.GetMailLabelUnreadCounts(characterID)
}

func GetMailListUnreadCounts(characterID int32) (map[int32]int, error) {
	return model.GetMailListUnreadCounts(characterID)
}

func ListMailLists(characterID int32) ([]EveEntity, error) {
	ll, err := model.ListMailLists(characterID)
	if err != nil {
		return nil, err
	}
	ee := make([]EveEntity, len(ll))
	for i, l := range ll {
		ee[i] = eveEntityFromDBModel(l.EveEntity)
	}
	return ee, nil
}

// ListMailsForLabel returns a character's mails for a label in descending order by timestamp.
// Return mails for all labels, when labelID = 0
func ListMailsForLabel(characterID int32, labelID int32) ([]Mail, error) {
	mm, err := model.ListMailsForLabel(characterID, labelID)
	if err != nil {
		return nil, err
	}
	mm2 := make([]Mail, len(mm))
	for i, m := range mm {
		mm2[i] = mailFromDBModel(m)
	}
	return mm2, nil
}

func ListMailsForList(characterID int32, listID int32) ([]Mail, error) {
	mm, err := model.ListMailsForList(characterID, listID)
	if err != nil {
		return nil, err
	}
	mm2 := make([]Mail, len(mm))
	for i, m := range mm {
		mm2[i] = mailFromDBModel(m)
	}
	return mm2, nil
}
