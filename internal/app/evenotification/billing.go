package evenotification

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

func (s *EveNotificationService) renderBilling(ctx context.Context, type_, text string) (optional.Optional[string], optional.Optional[string], error) {
	var title, body optional.Optional[string]
	switch type_ {
	case BillPaidCorpAllMsg:
		title.Set("Bill payed")
		var data notification.BillPaidCorpAllMsg
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out := fmt.Sprintf(
			"A bill of **%s** ISK, due **%s** was payed.",
			humanize.Commaf(float64(data.Amount)),
			fromLDAPTime(data.DueDate).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case BillOutOfMoneyMsg:
		title.Set("Insufficient Funds for Bill")
		var data notification2.CorpAllBillMsgV2
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		out := fmt.Sprintf(
			"The selected corporation wallet division for automatic payments "+
				"does not have enough current funds available to pay the %s bill, "+
				"due to be paid by %s. "+
				"Transfer additional funds to the selected wallet "+
				"division in order to meet your pending automatic bills.",
			billTypeName(data.BillTypeID),
			fromLDAPTime(data.DueDate).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case CorpAllBillMsg:
		title.Set("Bill issued")
		var data notification2.CorpAllBillMsgV2
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		entities, err := s.EveUniverseService.ToEveEntities(ctx, []int32{data.CreditorID, data.DebtorID})
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf(
			"A bill of **%s** ISK, due **%s** owed by %s to %s was issued on %s. This bill is for %s.",
			humanize.Commaf(data.Amount),
			fromLDAPTime(data.DueDate).Format(app.TimeDefaultFormat),
			makeEveEntityProfileLink(entities[data.DebtorID]),
			makeEveEntityProfileLink(entities[data.CreditorID]),
			fromLDAPTime(data.CurrentDate).Format(app.TimeDefaultFormat),
			billTypeName(data.BillTypeID),
		)
		body.Set(out)

	case InfrastructureHubBillAboutToExpire:
		title.Set("IHub Bill About to Expire")
		var data notification2.InfrastructureHubBillAboutToExpire
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		solarSystem, err := s.EveUniverseService.GetOrCreateEveSolarSystemESI(ctx, data.SolarSystemID)
		if err != nil {
			return title, body, err
		}
		out := fmt.Sprintf("Maintenance bill for Infrastructure Hub in %s expires at %s, "+
			"if not paid in time this Infrastructure Hub will self-destruct.",
			makeLocationLink(solarSystem),
			fromLDAPTime(data.DueDate).Format(app.TimeDefaultFormat),
		)
		body.Set(out)

	case IHubDestroyedByBillFailure:
		var data notification2.IHubDestroyedByBillFailure
		if err := yaml.Unmarshal([]byte(text), &data); err != nil {
			return title, body, err
		}
		solarSystem, err := s.EveUniverseService.GetOrCreateEveSolarSystemESI(ctx, data.SolarSystemID)
		if err != nil {
			return title, body, err
		}
		structureType, err := s.EveUniverseService.GetOrCreateEveTypeESI(ctx, int32(data.StructureTypeID))
		if err != nil {
			return title, body, err
		}
		title.Set(fmt.Sprintf(
			"%s has self-destructed due to unpaid maintenance bills",
			structureType.Name,
		))
		out := fmt.Sprintf("%s in %s has self-destructed, as the standard maintenance bills where not paid.",
			structureType.Name,
			makeLocationLink(solarSystem),
		)
		body.Set(out)
	}
	return title, body, nil
}

func billTypeName(id int32) string {
	switch id {
	case 7:
		return "Infrastructure Hub"
	}
	return "?"
}
