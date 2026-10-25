package app

import "maps"

// Division represents a division in an EVE Online corporation.
type Division uint

const (
	DivisionZero Division = iota
	Division1
	Division2
	Division3
	Division4
	Division5
	Division6
	Division7
)

var defaultWalletNames = map[Division]string{
	Division1: "Master Wallet",
	Division2: "2nd Wallet Division",
	Division3: "3rd Wallet Division",
	Division4: "4th Wallet Division",
	Division5: "5th Wallet Division",
	Division6: "6th Wallet Division",
	Division7: "7th Wallet Division",
}

func (d Division) DefaultWalletName() string {
	return defaultWalletNames[d]
}

// DefaultCorporationWalletNames returns the default wallet names of all divisions.
func DefaultCorporationWalletNames() map[Division]string {
	return maps.Clone(defaultWalletNames)
}

var defaultHangarNames = map[Division]string{
	Division1: "1st Division",
	Division2: "2nd Division",
	Division3: "3rd Division",
	Division4: "4th Division",
	Division5: "5th Division",
	Division6: "6th Division",
	Division7: "7th Division",
}

// DefaultHangarName returns the generic in-game name for a corporation hangar division,
// used when the corporation has not set a custom name for it.
func (d Division) DefaultHangarName() string {
	return defaultHangarNames[d]
}

var Divisions = []Division{
	Division1,
	Division2,
	Division3,
	Division4,
	Division5,
	Division6,
	Division7,
}

type Corporation struct {
	ID             int64
	EveCorporation *EveCorporation
}

func (c *Corporation) IDOrZero() int64 {
	if c == nil {
		return 0
	}
	return c.ID
}

func (c *Corporation) NameOrZero() string {
	if c == nil || c.EveCorporation == nil {
		return ""
	}
	return c.EveCorporation.Name
}

type CorporationHangarName struct {
	CorporationID int64
	Division      Division
	Name          string
}

// CorporationMember represents a member in an EVE Online corporation.
type CorporationMember struct {
	CorporationID int64
	Character     *EveEntity
}
