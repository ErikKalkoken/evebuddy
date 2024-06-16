package mailrecipients

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type mailRecipientCategory uint

const (
	mailRecipientCategoryUnknown mailRecipientCategory = iota
	mailRecipientCategoryAlliance
	mailRecipientCategoryCorporation
	mailRecipientCategoryCharacter
	mailRecipientCategoryMailList
)

var mailRecipientCategoryLabels = map[mailRecipientCategory]string{
	mailRecipientCategoryAlliance:    "Alliance",
	mailRecipientCategoryCorporation: "Corporation",
	mailRecipientCategoryCharacter:   "Character",
	mailRecipientCategoryMailList:    "Mailing List",
}

var mailRecipientMapCategories = map[app.EveEntityCategory]mailRecipientCategory{
	app.EveEntityAlliance:    mailRecipientCategoryAlliance,
	app.EveEntityCharacter:   mailRecipientCategoryCharacter,
	app.EveEntityCorporation: mailRecipientCategoryCorporation,
	app.EveEntityMailList:    mailRecipientCategoryMailList,
}

func (r mailRecipientCategory) String() string {
	return mailRecipientCategoryLabels[r]
}

// A recipient in a mail
type recipient struct {
	name     string
	category mailRecipientCategory
}

func newRecipientFromEntity(e *app.EveEntity) recipient {
	r := recipient{name: e.Name}
	c, ok := mailRecipientMapCategories[e.Category]
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
		for k, v := range mailRecipientCategoryLabels {
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
	return r.category != mailRecipientCategoryUnknown
}

func (r *recipient) eveEntityCategory() (app.EveEntityCategory, bool) {
	for ec, rc := range mailRecipientMapCategories {
		if rc == r.category {
			return ec, true
		}
	}
	return 0, false
}

func (r *recipient) empty() bool {
	return r.name == "" && !r.hasCategory()
}

// A container holding all recipients in a mail
type MailRecipients struct {
	list []recipient
}

func New() *MailRecipients {
	var rr MailRecipients
	return &rr
}

func NewFromEntities(ee []*app.EveEntity) *MailRecipients {
	rr := New()
	for _, e := range ee {
		o := newRecipientFromEntity(e)
		rr.list = append(rr.list, o)
	}
	return rr
}

func NewFromText(t string) *MailRecipients {
	rr := New()
	if t == "" {
		return rr
	}
	ss := strings.Split(t, ",")
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

func (rr *MailRecipients) AddFromEveEntity(e *app.EveEntity) {
	r := newRecipientFromEntity(e)
	rr.add(r)
}

func (rr *MailRecipients) AddFromText(s string) {
	r := newRecipientFromText(s)
	rr.add(r)
}

func (rr *MailRecipients) add(r recipient) {
	rr.list = append(rr.list, r)
}

func (rr *MailRecipients) Size() int {
	return len(rr.list)
}

func (rr *MailRecipients) String() string {
	var ss []string
	for _, e := range rr.list {
		ss = append(ss, e.String())
	}
	s := strings.Join(ss, ", ")
	return s
}

func (rr *MailRecipients) ToOptions() []string {
	ss := make([]string, len(rr.list))
	for i, r := range rr.list {
		ss[i] = r.String()
	}
	return ss
}

// Returns the mail recipients as unclean EveEntity slice.
// ID will not be set. And some might not have a category.
func (rr *MailRecipients) ToEveEntitiesUnclean() []*app.EveEntity {
	ee := make([]*app.EveEntity, len(rr.list))
	for i, r := range rr.list {
		e := app.EveEntity{Name: r.name}
		c, ok := r.eveEntityCategory()
		if ok {
			e.Category = c
		}
		ee[i] = &e
	}
	return ee
}
