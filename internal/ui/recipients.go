package ui

import (
	"errors"
	"example/esiapp/internal/api/esi"
	"example/esiapp/internal/model"
	"fmt"
	"regexp"
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

func (rr *recipients) ToEsiRecipients() ([]esi.MailRecipient, error) {
	mm, names, err := rr.resolveLocally()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return mm, nil
	}
	r, err := esi.ResolveEntityNames(httpClient, names)
	if err != nil {
		return nil, err
	}
	for _, o := range r.Alliances {
		mm = append(mm, esi.MailRecipient{ID: o.ID, Type: esi.MailRecipientTypeAlliance})
	}
	for _, o := range r.Characters {
		mm = append(mm, esi.MailRecipient{ID: o.ID, Type: esi.MailRecipientTypeCharacter})
	}
	for _, o := range r.Corporations {
		mm = append(mm, esi.MailRecipient{ID: o.ID, Type: esi.MailRecipientTypeCorporation})
	}
	return mm, nil
}

// resolveLocally tries to resolve recipients against known EveEntities.
// It returns resolved recipients and a list of unresolved names (if any)
func (rr *recipients) resolveLocally() ([]esi.MailRecipient, []string, error) {
	mm := make([]esi.MailRecipient, 0, len(rr.list))
	names := make([]string, 0, len(rr.list))
	for _, r := range rr.list {
		if r.hasCategory() {
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
	}
	return mm, names, nil
}
