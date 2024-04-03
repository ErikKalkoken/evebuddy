package ui

import (
	"errors"
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/model"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type recipientCategory uint

const (
	recipientCategoryUnknown recipientCategory = iota
	recipientCategoryAlliance
	recipientCategoryCorporation
	recipientCategoryCharacter
	recipientCategoryMailList
)

var ErrNameNoMatch = fmt.Errorf("recipients: no matching name")
var ErrNameMultipleMatches = fmt.Errorf("recipients: multiple matching names")

var recipientCategoryLabels = map[recipientCategory]string{
	recipientCategoryAlliance:    "Alliance",
	recipientCategoryCorporation: "Corporation",
	recipientCategoryCharacter:   "Character",
	recipientCategoryMailList:    "Mailing List",
}

var recipientMapCategories = map[model.EveEntityCategory]recipientCategory{
	model.EveEntityAlliance:    recipientCategoryAlliance,
	model.EveEntityCharacter:   recipientCategoryCharacter,
	model.EveEntityCorporation: recipientCategoryCorporation,
	model.EveEntityMailList:    recipientCategoryMailList,
}

func (r recipientCategory) String() string {
	return recipientCategoryLabels[r]
}

// A recipient in a mail
type recipient struct {
	name     string
	category recipientCategory
}

func newRecipientFromEntity(e model.EveEntity) recipient {
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

func (r *recipient) eveEntityCategory() (model.EveEntityCategory, bool) {
	for ec, rc := range recipientMapCategories {
		if rc == r.category {
			return ec, true
		}
	}
	return "", false
}

func (r *recipient) empty() bool {
	return r.name == "" && !r.hasCategory()
}

// All recipients in a mail
type recipients struct {
	list []recipient
}

func NewRecipients() *recipients {
	var rr recipients
	return &rr
}

func NewRecipientsFromEntities(ee []model.EveEntity) *recipients {
	rr := NewRecipients()
	for _, e := range ee {
		o := newRecipientFromEntity(e)
		rr.list = append(rr.list, o)
	}
	return rr
}

func NewRecipientsFromText(s string) *recipients {
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

func (rr *recipients) AddFromEveEntity(e model.EveEntity) {
	r := newRecipientFromEntity(e)
	rr.add(r)
}

func (rr *recipients) AddFromText(s string) {
	r := newRecipientFromText(s)
	rr.add(r)
}

func (rr *recipients) add(r recipient) {
	rr.list = append(rr.list, r)
}

func (rr *recipients) Size() int {
	return len(rr.list)
}

func (rr *recipients) String() string {
	var ss []string
	for _, e := range rr.list {
		ss = append(ss, e.String())
	}
	s := strings.Join(ss, ", ")
	return s
}

func (rr *recipients) ToOptions() []string {
	ss := make([]string, len(rr.list))
	for i, r := range rr.list {
		ss[i] = r.String()
	}
	return ss
}

var eveEntityCategory2MailRecipientType = map[model.EveEntityCategory]esi.MailRecipientType{
	model.EveEntityAlliance:    esi.MailRecipientTypeAlliance,
	model.EveEntityCharacter:   esi.MailRecipientTypeCharacter,
	model.EveEntityCorporation: esi.MailRecipientTypeCorporation,
	model.EveEntityMailList:    esi.MailRecipientTypeMailingList,
}

func (rr *recipients) ToMailRecipients() ([]esi.MailRecipient, error) {
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
func (rr *recipients) buildMailRecipients() ([]esi.MailRecipient, []string, error) {
	mm := make([]esi.MailRecipient, 0, len(rr.list))
	names := make([]string, 0, len(rr.list))
	for _, r := range rr.list {
		category, ok := r.eveEntityCategory()
		if !ok {
			names = append(names, r.name)
			continue
		}
		entity, err := model.FetchEveEntityByNameAndCategory(r.name, category)
		if err != nil {
			if errors.Is(err, model.ErrDoesNotExist) {
				names = append(names, r.name)
				continue
			} else {
				return nil, nil, err
			}
		}
		mailType, ok := eveEntityCategory2MailRecipientType[entity.Category]
		if !ok {
			names = append(names, r.name)
			continue
		}
		m := esi.MailRecipient{ID: entity.ID, Type: mailType}
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
	r, err := esi.ResolveEntityNames(httpClient, names)
	if err != nil {
		return nil, err
	}
	for _, o := range r.Alliances {
		e := model.EveEntity{ID: o.ID, Name: o.Name, Category: model.EveEntityAlliance}
		ee = append(ee, e)
	}
	for _, o := range r.Characters {
		e := model.EveEntity{ID: o.ID, Name: o.Name, Category: model.EveEntityCharacter}
		ee = append(ee, e)
	}
	for _, o := range r.Corporations {
		e := model.EveEntity{ID: o.ID, Name: o.Name, Category: model.EveEntityCorporation}
		ee = append(ee, e)
	}
	return ee, nil
}

// buildMailRecipientsFromNames tries to build MailRecipient objects from given names
// by checking against EveEntity objects in the database.
// Will abort with errors if no match is found or if multiple matches are found for a name.
func buildMailRecipientsFromNames(names []string) ([]esi.MailRecipient, error) {
	mm := make([]esi.MailRecipient, 0, len(names))
	for _, n := range names {
		ee, err := model.FindEveEntitiesByName(n)
		if err != nil {
			return nil, err
		}
		if len(ee) == 0 {
			return nil, fmt.Errorf("for name %s: %w", n, ErrNameNoMatch)
		}
		if len(ee) > 1 {
			return nil, fmt.Errorf("for name %s: %w", n, ErrNameMultipleMatches)
		}
		e := ee[0]
		c, ok := eveEntityCategory2MailRecipientType[e.Category]
		if !ok {
			return nil, fmt.Errorf("failed to match category for entity: %v", e)
		}
		m := esi.MailRecipient{ID: e.ID, Type: c}
		mm = append(mm, m)
	}
	return mm, nil
}
