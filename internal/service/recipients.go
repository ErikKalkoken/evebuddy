package service

import (
	"context"
	"database/sql"
	"errors"
	"example/evebuddy/internal/model"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/antihax/goesi/esi"
)

type recipientCategory uint

const (
	recipientCategoryUnknown recipientCategory = iota
	recipientCategoryAlliance
	recipientCategoryCorporation
	recipientCategoryCharacter
	recipientCategoryMailList
)

var ErrNameNoMatch = errors.New("no matching name")
var ErrNameMultipleMatches = errors.New("multiple matching names")

var recipientCategoryLabels = map[recipientCategory]string{
	recipientCategoryAlliance:    "Alliance",
	recipientCategoryCorporation: "Corporation",
	recipientCategoryCharacter:   "Character",
	recipientCategoryMailList:    "Mailing List",
}

var recipientMapCategories = map[EveEntityCategory]recipientCategory{
	EveEntityAlliance:    recipientCategoryAlliance,
	EveEntityCharacter:   recipientCategoryCharacter,
	EveEntityCorporation: recipientCategoryCorporation,
	EveEntityMailList:    recipientCategoryMailList,
}

var eveEntityCategory2MailRecipientType = map[EveEntityCategory]string{
	EveEntityAlliance:    "alliance",
	EveEntityCharacter:   "character",
	EveEntityCorporation: "corporation",
	EveEntityMailList:    "mailing_list",
}

func (r recipientCategory) String() string {
	return recipientCategoryLabels[r]
}

// A recipient in a mail
type recipient struct {
	name     string
	category recipientCategory
}

func newRecipientFromEntity(e EveEntity) recipient {
	r := recipient{name: e.Name}
	c, ok := recipientMapCategories[e.Category]
	if ok {
		r.category = c
	}
	return r
}

func newRecipientFromText(s string) recipient {
	re, _ := regexp.Compile(`^([^\[\]]+)( \[(.*)\])?$`)
	m := re.FindStringSubmatch(s)
	var r recipient
	if len(m) >= 1 && m[1] != "" {
		r.name = m[1]
	}
	if len(m) >= 3 && m[3] != "" {
		for k, v := range recipientCategoryLabels {
			if v == m[3] {
				r.category = k
				break
			}
		}
	}
	return r
}

func (r *recipient) String() string {
	if !r.hasCategory() {
		return r.name
	}
	s := fmt.Sprintf("%s [%s]", r.name, r.category)
	return s
}

func (r *recipient) hasCategory() bool {
	return r.category != recipientCategoryUnknown
}

func (r *recipient) eveEntityCategory() (EveEntityCategory, bool) {
	for ec, rc := range recipientMapCategories {
		if rc == r.category {
			return ec, true
		}
	}
	return 0, false
}

func (r *recipient) empty() bool {
	return r.name == "" && !r.hasCategory()
}

// A container holding all Recipients in a mail
type Recipients struct {
	list []recipient
}

func NewRecipients() *Recipients {
	var rr Recipients
	return &rr
}

func NewRecipientsFromEntities(ee []EveEntity) *Recipients {
	rr := NewRecipients()
	for _, e := range ee {
		o := newRecipientFromEntity(e)
		rr.list = append(rr.list, o)
	}
	return rr
}

func NewRecipientsFromText(s string) *Recipients {
	rr := NewRecipients()
	if s == "" {
		return rr
	}
	ss := strings.Split(s, ",")
	for i, x := range ss {
		ss[i] = strings.Trim(x, " ")
	}
	for _, s := range ss {
		r := newRecipientFromText(s)
		if r.empty() {
			continue
		}
		rr.add(r)
	}
	return rr
}

func (rr *Recipients) AddFromEveEntity(e EveEntity) {
	r := newRecipientFromEntity(e)
	rr.add(r)
}

func (rr *Recipients) AddFromText(s string) {
	r := newRecipientFromText(s)
	rr.add(r)
}

func (rr *Recipients) add(r recipient) {
	rr.list = append(rr.list, r)
}

func (rr *Recipients) Size() int {
	return len(rr.list)
}

func (rr *Recipients) String() string {
	var ss []string
	for _, e := range rr.list {
		ss = append(ss, e.String())
	}
	s := strings.Join(ss, ", ")
	return s
}

func (rr *Recipients) ToOptions() []string {
	ss := make([]string, len(rr.list))
	for i, r := range rr.list {
		ss[i] = r.String()
	}
	return ss
}

func (rr *Recipients) ToMailRecipients() ([]esi.PostCharactersCharacterIdMailRecipient, error) {
	mm1, names, err := rr.buildMailRecipients()
	if err != nil {
		return nil, err
	}
	ee, err := resolveNamesRemotely(names)
	if err != nil {
		return nil, err
	}
	for _, e := range ee {
		if err := e.Save(); err != nil {
			return nil, err
		}
	}
	mm2, err := buildMailRecipientsFromNames(names)
	if err != nil {
		return nil, err
	}
	mm := slices.Concat(mm1, mm2)
	return mm, nil
}

// buildMailRecipients tries to build MailRecipients from recipients.
// It returns resolved recipients and a list of remaining unresolved names (if any)
func (rr *Recipients) buildMailRecipients() ([]esi.PostCharactersCharacterIdMailRecipient, []string, error) {
	mm := make([]esi.PostCharactersCharacterIdMailRecipient, 0, len(rr.list))
	names := make([]string, 0, len(rr.list))
	for _, r := range rr.list {
		c, ok := r.eveEntityCategory()
		if !ok {
			names = append(names, r.name)
			continue
		}
		e, err := model.GetEveEntityByNameAndCategory(r.name, eveEntityDBModelCategoryFromCategory(c))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				names = append(names, r.name)
				continue
			} else {
				return nil, nil, err
			}
		}
		mailType, ok := eveEntityCategory2MailRecipientType[eveEntityCategoryFromDBModel(e.Category)]
		if !ok {
			names = append(names, r.name)
			continue
		}
		m := esi.PostCharactersCharacterIdMailRecipient{RecipientId: e.ID, RecipientType: mailType}
		mm = append(mm, m)
	}
	return mm, names, nil
}

// resolveNamesRemotely resolves a list of names remotely and returns all matches.
func resolveNamesRemotely(names []string) ([]model.EveEntity, error) {
	ee := make([]model.EveEntity, 0, len(names))
	if len(names) == 0 {
		return ee, nil
	}
	r, _, err := esiClient.ESI.UniverseApi.PostUniverseIds(context.Background(), names, nil)
	if err != nil {
		return nil, err
	}
	for _, o := range r.Alliances {
		e := model.EveEntity{ID: o.Id, Name: o.Name, Category: model.EveEntityAlliance}
		ee = append(ee, e)
	}
	for _, o := range r.Characters {
		e := model.EveEntity{ID: o.Id, Name: o.Name, Category: model.EveEntityCharacter}
		ee = append(ee, e)
	}
	for _, o := range r.Corporations {
		e := model.EveEntity{ID: o.Id, Name: o.Name, Category: model.EveEntityCorporation}
		ee = append(ee, e)
	}
	return ee, nil
}

// buildMailRecipientsFromNames tries to build MailRecipient objects from given names
// by checking against EveEntity objects in the database.
// Will abort with errors if no match is found or if multiple matches are found for a name.
func buildMailRecipientsFromNames(names []string) ([]esi.PostCharactersCharacterIdMailRecipient, error) {
	mm := make([]esi.PostCharactersCharacterIdMailRecipient, 0, len(names))
	for _, n := range names {
		ee, err := model.ListEveEntitiesByName(n)
		if err != nil {
			return nil, err
		}
		if len(ee) == 0 {
			return nil, fmt.Errorf("%s: %w", n, ErrNameNoMatch)
		}
		if len(ee) > 1 {
			return nil, fmt.Errorf("%s: %w", n, ErrNameMultipleMatches)
		}
		e := ee[0]
		c, ok := eveEntityCategory2MailRecipientType[eveEntityCategoryFromDBModel(e.Category)]
		if !ok {
			return nil, fmt.Errorf("failed to match category for entity: %v", e)
		}
		m := esi.PostCharactersCharacterIdMailRecipient{RecipientId: e.ID, RecipientType: c}
		mm = append(mm, m)
	}
	return mm, nil
}
