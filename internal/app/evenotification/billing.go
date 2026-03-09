package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

const (
	billTypeLease             = 2
	billTypeAlliance          = 5
	billTypeInfrastructureHub = 7
)

func billTypeName(id int64) string {
	switch id {
	case billTypeLease:
		return "lease"
	case billTypeAlliance:
		return "alliance maintenance"
	case billTypeInfrastructureHub:
		return "infrastructure hub upkeep"
	}
	return "?"
}

type billOutOfMoneyMsg struct {
	baseRenderer
}

func (n billOutOfMoneyMsg) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	var data goesi.CorpAllBillMsgV2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Insufficient funds for %s bill", billTypeName(data.BillTypeID))
	out := fmt.Sprintf(
		"The selected corporation wallet division for automatic payments "+
			"does not have enough current funds available to pay the %s bill, "+
			"due to be paid by %s. "+
			"Transfer additional funds to the selected wallet "+
			"division in order to meet your pending automatic bills.",
		billTypeName(data.BillTypeID),
		fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
	)
	body = out
	return title, body, nil
}

type billPaidCorpAllMsg struct {
	baseRenderer
}

func (n billPaidCorpAllMsg) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	title = "Bill payed"
	var data goesi.BillPaidCorpAllMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	out := fmt.Sprintf(
		"A bill of **%s** ISK, due **%s** was payed.",
		humanize.Commaf(float64(data.Amount)),
		fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
	)
	body = out
	return title, body, nil
}

type corpAllBillMsg struct {
	baseRenderer
}

func (n corpAllBillMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n corpAllBillMsg) unmarshal(text string) (goesi.CorpAllBillMsgV2, set.Set[int64], error) {
	var data goesi.CorpAllBillMsgV2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CreditorID, data.DebtorID)
	if data.ExternalID != 0 && data.ExternalID != -1 && data.ExternalID == int64(int32(data.ExternalID)) {
		ids.Add(data.ExternalID)
	}
	if data.ExternalID2 != 0 && data.ExternalID2 != -1 && data.ExternalID2 == int64(int32(data.ExternalID2)) {
		ids.Add(data.ExternalID2)
	}
	return data, ids, nil
}

func (n corpAllBillMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	var external1 string
	if x, ok := entities[data.ExternalID]; ok && x.Name != "" {
		external1 = x.Name
	} else {
		external1 = "?"
	}
	var external2 string
	if x, ok := entities[data.ExternalID2]; ok && x.Name != "" {
		external2 = x.Name
	} else {
		external2 = "?"
	}
	var billPurpose string
	switch data.BillTypeID {
	case billTypeLease:
		billPurpose = fmt.Sprintf("extending the lease of **%s** at **%s**", external1, external2)
	case billTypeAlliance:
		billPurpose = fmt.Sprintf("maintenance of **%s**", external1)
	case billTypeInfrastructureHub:
		billPurpose = fmt.Sprintf("maintenance of infrastructure hub in **%s**", external1)
	default:
		billPurpose = "?"
	}
	body = fmt.Sprintf(
		"A bill of **%s** ISK, due **%s** owed by %s to %s was issued on %s. This bill is for %s.",
		humanize.Commaf(data.Amount),
		fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
		makeEveEntityProfileLink(entities[data.DebtorID]),
		makeEveEntityProfileLink(entities[data.CreditorID]),
		fromLDAPTime(data.CurrentDate).Format(app.DateTimeFormat),
		billPurpose,
	)
	title = fmt.Sprintf("Bill issued for %s", billTypeName(data.BillTypeID))
	return title, body, err
}

type infrastructureHubBillAboutToExpire struct {
	baseRenderer
}

func (n infrastructureHubBillAboutToExpire) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	title = "IHub Bill About to Expire"
	var data goesi.InfrastructureHubBillAboutToExpire
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	out := fmt.Sprintf("Maintenance bill for Infrastructure Hub in %s expires at %s, "+
		"if not paid in time this Infrastructure Hub will self-destruct.",
		makeSolarSystemLink(solarSystem),
		fromLDAPTime(data.DueDate).Format(app.DateTimeFormat),
	)
	body = out
	return title, body, nil
}

type iHubDestroyedByBillFailure struct {
	baseRenderer
}

func (n iHubDestroyedByBillFailure) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	var data goesi.IHubDestroyedByBillFailure
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	structureType, err := n.eus.GetOrCreateTypeESI(ctx, int64(data.StructureTypeID))
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf(
		"%s has self-destructed due to unpaid maintenance bills",
		structureType.Name,
	)
	out := fmt.Sprintf("%s in %s has self-destructed, as the standard maintenance bills where not paid.",
		structureType.Name,
		makeSolarSystemLink(solarSystem),
	)
	body = out
	return title, body, nil
}
