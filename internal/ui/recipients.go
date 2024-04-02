package ui

import (
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

func NewRecipientFromEntity(e model.EveEntity) recipient {
	r := recipient{name: e.Name}
	c, ok := recipientMapCategories[e.Category]
	if ok {
		r.category = c
	}
	return r
}

func NewRecipientFromText(s string) recipient {
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
	if r.category == recipientCategoryUnknown {
		return r.name
	}
	s := fmt.Sprintf("%s [%s]", r.name, r.category)
	return s
}

func (r *recipient) Empty() bool {
	return r.name == "" && r.category == recipientCategoryUnknown
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
		o := NewRecipientFromEntity(e)
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
		r := NewRecipientFromText(s)
		if r.Empty() {
			continue
		}
		rr.add(r)
	}
	return rr
}

func (rr *recipients) String() string {
	var ss []string
	for _, e := range rr.list {
		ss = append(ss, e.String())
	}
	s := strings.Join(ss, ", ")
	return s
}

func (rr *recipients) add(r recipient) {
	rr.list = append(rr.list, r)
}

func (rr *recipients) names() []string {
	ss := make([]string, len(rr.list))
	for i, r := range rr.list {
		ss[i] = r.name
	}
	return ss
}

func (rr *recipients) Size() int {
	return len(rr.list)
}
