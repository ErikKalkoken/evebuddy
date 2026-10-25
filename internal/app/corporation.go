package app

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

// TODO: Move ID resolution into storage layer

func (d Division) ID() int64 {
	return int64(d)
}

func (d Division) DefaultWalletName() string {
	m := map[Division]string{
		Division1: "Master Wallet",
		Division2: "2nd Wallet",
		Division3: "3rd Wallet",
		Division4: "4th Wallet",
		Division5: "5th Wallet",
		Division6: "6th Wallet",
		Division7: "7th Wallet",
	}
	return m[d]
}

// DefaultHangarName returns the generic in-game name for a corporation hangar division,
// used when the corporation has not set a custom name for it.
func (d Division) DefaultHangarName() string {
	m := map[Division]string{
		Division1: "1st Division",
		Division2: "2nd Division",
		Division3: "3rd Division",
		Division4: "4th Division",
		Division5: "5th Division",
		Division6: "6th Division",
		Division7: "7th Division",
	}
	return m[d]
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
	DivisionID    int64
	Name          string
}

// CorporationMember represents a member in an EVE Online corporation.
type CorporationMember struct {
	CorporationID int64
	Character     *EveEntity
}
