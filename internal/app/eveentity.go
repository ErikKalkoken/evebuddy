package app

import (
	"cmp"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// An EveEntity in EveOnline.
type EveEntity struct {
	Category EveEntityCategory
	ID       int32
	Name     string
}

func (ee EveEntity) CategoryDisplay() string {
	titler := cases.Title(language.English)
	return titler.String(ee.Category.String())
}

func (ee EveEntity) IsCharacter() bool {
	return ee.Category == EveEntityCharacter
}

func (ee *EveEntity) Compare(other *EveEntity) int {
	return cmp.Compare(ee.Name, other.Name)
}
