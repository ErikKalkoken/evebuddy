package evenotification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification/notification2"
)

type allianceCapitalChanged struct {
	baseRenderer
}

func (n allianceCapitalChanged) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allianceCapitalChanged) unmarshal(text string) (goesi.AllianceCapitalChanged, set.Set[int64], error) {
	var data goesi.AllianceCapitalChanged
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID)
	return data, ids, nil
}

func (n allianceCapitalChanged) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Capital system changed for %s", entities[data.AllianceID].Name)
	body = fmt.Sprintf(
		"The capital system of %s has been changed to %s.",
		makeEveEntityProfileLink(entities[data.AllianceID]),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type buddyConnectContactAdd struct {
	baseRenderer
}

func (n buddyConnectContactAdd) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.BuddyConnectContactAdd
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "New contact added"
	body := fmt.Sprintf(
		"A new contact has been added at standing level **%d**.\n\n> %s",
		data.Level,
		data.Message,
	)
	return title, body, nil
}

type contactAdd struct {
	baseRenderer
}

func (n contactAdd) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.ContactAdd
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "New contact added"
	body := fmt.Sprintf(
		"A new contact has been added at standing level **%d**.\n\n> %s",
		data.Level,
		data.Message,
	)
	return title, body, nil
}

type contactEdit struct {
	baseRenderer
}

func (n contactEdit) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.ContactEdit
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Contact standing updated"
	body := fmt.Sprintf(
		"A contact's standing has been updated to **%.1f**.\n\n> %s",
		data.Level,
		data.Message,
	)
	return title, body, nil
}

type containerPasswordMsg struct {
	baseRenderer
}

func (n containerPasswordMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n containerPasswordMsg) unmarshal(text string) (goesi.ContainerPasswordMsg, set.Set[int64], error) {
	var data goesi.ContainerPasswordMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID, data.TypeID)
	return data, ids, nil
}

func (n containerPasswordMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Container password accessed in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"%s has accessed a **%s** container in %s using the **%s** password.",
		makeEveEntityProfileLink(entities[data.CharID]),
		entities[data.TypeID].Name,
		makeSolarSystemLink(solarSystem),
		data.PasswordType,
	)
	return title, body, nil
}

type customsMsg struct {
	baseRenderer
}

func (n customsMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.CustomsMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Customs notification in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"You have received a customs notification in %s. "+
			"Security level: **%.2f**.",
		makeSolarSystemLink(solarSystem),
		data.SecurityLevel,
	)
	return title, body, nil
}

type gameTimeAdded struct {
	baseRenderer
}

func (n gameTimeAdded) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Game time added"
	body := "Game time has been added to your account."
	return title, body, nil
}

type gameTimeReceived struct {
	baseRenderer
}

func (n gameTimeReceived) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n gameTimeReceived) unmarshal(text string) (goesi.GameTimeReceived, set.Set[int64], error) {
	var data goesi.GameTimeReceived
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.SenderCharID)
	return data, ids, nil
}

func (n gameTimeReceived) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Game time received"
	body = fmt.Sprintf(
		"You have received **%d** days of game time from %s.\n\n> %s",
		data.Quantity,
		makeEveEntityProfileLink(entities[data.SenderCharID]),
		data.Message,
	)
	return title, body, nil
}

type gameTimeSent struct {
	baseRenderer
}

func (n gameTimeSent) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n gameTimeSent) unmarshal(text string) (goesi.GameTimeSent, set.Set[int64], error) {
	var data goesi.GameTimeSent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.ReceiverCharID, data.SenderCharID)
	return data, ids, nil
}

func (n gameTimeSent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Game time sent"
	body = fmt.Sprintf(
		"%s has sent game time to %s.",
		makeEveEntityProfileLink(entities[data.SenderCharID]),
		makeEveEntityProfileLink(entities[data.ReceiverCharID]),
	)
	return title, body, nil
}

type giftReceived struct {
	baseRenderer
}

func (n giftReceived) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n giftReceived) unmarshal(text string) (goesi.GiftReceived, set.Set[int64], error) {
	var data goesi.GiftReceived
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.SenderCharID)
	return data, ids, nil
}

func (n giftReceived) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Gift received"
	body = fmt.Sprintf(
		"You have received a gift of **%d** units from %s.\n\n> %s",
		data.Quantity,
		makeEveEntityProfileLink(entities[data.SenderCharID]),
		data.Message,
	)
	return title, body, nil
}

type incursionCompletedMsg struct {
	baseRenderer
}

func (n incursionCompletedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.IncursionCompletedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Incursion completed in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"The Sansha incursion in %s has been completed.",
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type locateCharMsg struct {
	baseRenderer
}

func (n locateCharMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n locateCharMsg) unmarshal(text string) (goesi.LocateCharMsg, set.Set[int64], error) {
	var data goesi.LocateCharMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharacterID)
	return data, ids, nil
}

func (n locateCharMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Location report for %s", entities[data.CharacterID].Name)
	var location string
	if data.TargetLocation.SolarSystem != 0 {
		ss, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.TargetLocation.SolarSystem)
		if err == nil {
			location = makeSolarSystemLink(ss)
		}
	}
	if location == "" {
		location = "unknown location"
	}
	body = fmt.Sprintf(
		"An agent has reported the location of %s: **%s**.",
		makeEveEntityProfileLink(entities[data.CharacterID]),
		location,
	)
	return title, body, nil
}

type missionOfferExpirationMsg struct {
	baseRenderer
}

func (n missionOfferExpirationMsg) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.MissionOfferExpirationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Mission offer expiring"
	var headerParts []string
	for _, h := range data.Header {
		headerParts = append(headerParts, h)
	}
	headerText := strings.Join(headerParts, " ")
	if headerText == "" {
		headerText = "A mission offer is expiring soon."
	}
	body := headerText
	return title, body, nil
}

type npcStandingsGained struct {
	baseRenderer
}

func (n npcStandingsGained) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.NPCStandingsGained
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "NPC standings improved"
	body := fmt.Sprintf("Your standings with **%d** NPC factions/corps have improved.", len(data))
	return title, body, nil
}

type npcStandingsLost struct {
	baseRenderer
}

func (n npcStandingsLost) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.NPCStandingsLost
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "NPC standings decreased"
	body := fmt.Sprintf("Your standings with **%d** NPC factions/corps have decreased.", len(data))
	return title, body, nil
}

type oldLscMessages struct {
	baseRenderer
}

func (n oldLscMessages) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.OldLscMessages
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := data.Subject
	if title == "" {
		title = "Legacy message"
	}
	body := data.Body
	if body == "" {
		body = "No message body available."
	}
	return title, body, nil
}

type operationFinished struct {
	baseRenderer
}

func (n operationFinished) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.OperationFinished
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Cooperative operation finished"
	body := fmt.Sprintf(
		"A cooperative operation has been completed. Reward: **%d** ISK.",
		data.Rewards.Isk,
	)
	return title, body, nil
}

type reimbursementMsg struct {
	baseRenderer
}

func (n reimbursementMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.ReimbursementMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	shipType, err := n.eus.GetOrCreateEntityESI(ctx, data.ShipTypeID)
	if err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Ship reimbursement for %s", shipType.Name)
	body := fmt.Sprintf(
		"You have been reimbursed for the loss of your **%s** in %s.",
		shipType.Name,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type researchMissionAvailableMsg struct {
	baseRenderer
}

func (n researchMissionAvailableMsg) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Research mission available"
	body := "A new research mission is available from your agent."
	return title, body, nil
}

type seasonalChallengeCompleted struct {
	baseRenderer
}

func (n seasonalChallengeCompleted) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.SeasonalChallengeCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Seasonal challenge completed"
	body := fmt.Sprintf(
		"You have completed a seasonal challenge and earned **%d** points.",
		data.PointsAwarded,
	)
	return title, body, nil
}

type industryTeamAuctionLost struct {
	baseRenderer
}

func (n industryTeamAuctionLost) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.IndustryTeamAuctionLost
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Industry team auction lost in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"You have lost the industry team auction in %s. "+
			"Your bid was **%s** ISK out of a total bid of **%s** ISK.",
		makeSolarSystemLink(solarSystem),
		humanize.Commaf(data.YourAmount),
		humanize.Commaf(data.TotalIsk),
	)
	return title, body, nil
}

type industryTeamAuctionWon struct {
	baseRenderer
}

func (n industryTeamAuctionWon) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.IndustryTeamAuctionWon
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Industry team auction won in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"You have won the industry team auction in %s. "+
			"Your bid was **%s** ISK out of a total bid of **%s** ISK.",
		makeSolarSystemLink(solarSystem),
		humanize.Commaf(data.YourAmount),
		humanize.Commaf(data.TotalIsk),
	)
	return title, body, nil
}

type industryOperationFinished struct {
	baseRenderer
}

func (n industryOperationFinished) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.IndustryOperationFinished
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	productType, err := n.eus.GetOrCreateEntityESI(ctx, data.ProductTypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Industry job completed: %s", productType.Name)
	body := fmt.Sprintf(
		"Your industry job producing **%s** (**%d** runs) in %s has completed.",
		productType.Name,
		data.Runs,
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type invasionCompletedMsg struct {
	baseRenderer
}

func (n invasionCompletedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.InvasionCompletedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Triglavian invasion completed in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"The Triglavian invasion of %s has been completed.",
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type invasionSystemLogin struct {
	baseRenderer
}

func (n invasionSystemLogin) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.InvasionSystemLogin
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Entering Triglavian invasion system %s", solarSystem.Name)
	body := fmt.Sprintf(
		"You have entered %s, a system under Triglavian invasion.",
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type invasionSystemStart struct {
	baseRenderer
}

func (n invasionSystemStart) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.InvasionSystemStart
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Triglavian invasion started in %s", solarSystem.Name)
	body := fmt.Sprintf(
		"A Triglavian invasion has started in %s.",
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type lpAutoRedeemed struct {
	baseRenderer
}

func (n lpAutoRedeemed) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n lpAutoRedeemed) unmarshal(text string) (notification2.LPAutoRedeemed, set.Set[int64], error) {
	var data notification2.LPAutoRedeemed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.FactionID)
	return data, ids, nil
}

func (n lpAutoRedeemed) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Loyalty points auto-redeemed"
	body = fmt.Sprintf(
		"**%d** loyalty points from %s have been automatically redeemed.",
		data.LP,
		makeEveEntityProfileLink(entities[data.FactionID]),
	)
	return title, body, nil
}

type missionCanceledTriglavian struct {
	baseRenderer
}

func (n missionCanceledTriglavian) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.MissionCanceledTriglavian
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	agent, err := n.eus.GetOrCreateEntityESI(ctx, data.AgentID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Triglavian mission cancelled by %s", agent.Name)
	body := fmt.Sprintf(
		"A mission from Triglavian agent **%s** has been cancelled.",
		agent.Name,
	)
	return title, body, nil
}

type missionTimeoutMsg struct {
	baseRenderer
}

func (n missionTimeoutMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.MissionTimeoutMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	agent, err := n.eus.GetOrCreateEntityESI(ctx, data.AgentID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Mission timed out for agent %s", agent.Name)
	body := fmt.Sprintf(
		"Your mission from agent **%s** has timed out.",
		agent.Name,
	)
	return title, body, nil
}

type raffleCreated struct {
	baseRenderer
}

func (n raffleCreated) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.RaffleCreated
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	itemType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Raffle created: %s", itemType.Name)
	body := fmt.Sprintf(
		"A new raffle has been created for a **%s**.",
		itemType.Name,
	)
	return title, body, nil
}

type raffleExpired struct {
	baseRenderer
}

func (n raffleExpired) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.RaffleExpired
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	itemType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Raffle expired: %s", itemType.Name)
	body := fmt.Sprintf(
		"The raffle for a **%s** has expired.",
		itemType.Name,
	)
	return title, body, nil
}

type raffleFinished struct {
	baseRenderer
}

func (n raffleFinished) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.RaffleFinished
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	itemType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Raffle finished: %s", itemType.Name)
	var body string
	if data.IsWinner {
		body = fmt.Sprintf(
			"You won the raffle for a **%s**! Congratulations!",
			itemType.Name,
		)
	} else {
		body = fmt.Sprintf(
			"The raffle for a **%s** has finished.",
			itemType.Name,
		)
	}
	return title, body, nil
}

type spAutoRedeemed struct {
	baseRenderer
}

func (n spAutoRedeemed) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.SPAutoRedeemed
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Skill points auto-redeemed"
	body := fmt.Sprintf(
		"**%d** skill points have been automatically redeemed.",
		data.Amount,
	)
	return title, body, nil
}

type skinSequencingCompleted struct {
	baseRenderer
}

func (n skinSequencingCompleted) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.SkinSequencingCompleted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	skinType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("SKIN sequencing completed: %s", skinType.Name)
	body := fmt.Sprintf(
		"SKIN sequencing for **%s** has been completed.",
		skinType.Name,
	)
	return title, body, nil
}

type storyLineMissionAvailableMsg struct {
	baseRenderer
}

func (n storyLineMissionAvailableMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.StoryLineMissionAvailableMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	agent, err := n.eus.GetOrCreateEntityESI(ctx, data.AgentID)
	if err != nil {
		return "", "", err
	}
	title := "Story line mission available"
	body := fmt.Sprintf(
		"A new story line mission is available from agent **%s**.",
		agent.Name,
	)
	return title, body, nil
}

type transactionReversalMsg struct {
	baseRenderer
}

func (n transactionReversalMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n transactionReversalMsg) unmarshal(text string) (notification2.TransactionReversalMsg, set.Set[int64], error) {
	var data notification2.TransactionReversalMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n transactionReversalMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Transaction reversed"
	body = fmt.Sprintf(
		"A transaction of **%s** ISK involving %s has been reversed.",
		humanize.Commaf(data.Amount),
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type tutorialMsg struct {
	baseRenderer
}

func (n tutorialMsg) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Tutorial notification"
	body := "You have received a tutorial notification."
	return title, body, nil
}

type allAnchoringMsg struct {
	baseRenderer
}

func (n allAnchoringMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n allAnchoringMsg) unmarshal(text string) (notification2.AllAnchoringMsg, set.Set[int64], error) {
	var data notification2.AllAnchoringMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AllianceID, data.CorpID)
	return data, ids, nil
}

func (n allAnchoringMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Structure anchoring in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"%s of %s is anchoring a structure in %s.",
		makeEveEntityProfileLink(entities[data.CorpID]),
		makeEveEntityProfileLink(entities[data.AllianceID]),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type battlPunishFriendlyFire struct {
	baseRenderer
}

func (n battlPunishFriendlyFire) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n battlPunishFriendlyFire) unmarshal(text string) (notification2.BattlePunishFriendlyFire, set.Set[int64], error) {
	var data notification2.BattlePunishFriendlyFire
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.KilledCharID, data.KillerCharID)
	return data, ids, nil
}

func (n battlPunishFriendlyFire) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "Friendly fire fine issued"
	body = fmt.Sprintf(
		"%s has been fined **%s** ISK for killing %s (friendly fire).",
		makeEveEntityProfileLink(entities[data.KillerCharID]),
		humanize.Commaf(data.BillAmount),
		makeEveEntityProfileLink(entities[data.KilledCharID]),
	)
	return title, body, nil
}

type combatOperationFinished struct {
	baseRenderer
}

func (n combatOperationFinished) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.CombatOperationFinished
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Combat operation finished"
	body := "A combat operation has been completed."
	return title, body, nil
}

type contractRegionChangedToPochven struct {
	baseRenderer
}

func (n contractRegionChangedToPochven) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.ContractRegionChangedToPochven
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Contract region changed to Pochven"
	body := "One of your contracts has been moved to the Pochven region due to Triglavian conquest."
	return title, body, nil
}

type districtAttacked struct {
	baseRenderer
}

func (n districtAttacked) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n districtAttacked) unmarshal(text string) (notification2.DistrictAttacked, set.Set[int64], error) {
	var data notification2.DistrictAttacked
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.AggressorID)
	return data, ids, nil
}

func (n districtAttacked) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("District under attack in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"A district in %s is being attacked by %s.",
		makeSolarSystemLink(solarSystem),
		makeEveEntityProfileLink(entities[data.AggressorID]),
	)
	return title, body, nil
}

type dustAppAcceptedMsg struct {
	baseRenderer
}

func (n dustAppAcceptedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n dustAppAcceptedMsg) unmarshal(text string) (notification2.DustAppAcceptedMsg, set.Set[int64], error) {
	var data notification2.DustAppAcceptedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n dustAppAcceptedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	title = "DUST 514 application accepted"
	body = fmt.Sprintf(
		"The DUST 514 application for %s has been accepted.",
		makeEveEntityProfileLink(entities[data.CharID]),
	)
	return title, body, nil
}

type essMainBankLink struct {
	baseRenderer
}

func (n essMainBankLink) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return set.Set[int64]{}, err
	}
	return ids, nil
}

func (n essMainBankLink) unmarshal(text string) (notification2.ESSMainBankLink, set.Set[int64], error) {
	var data notification2.ESSMainBankLink
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.CharID)
	return data, ids, nil
}

func (n essMainBankLink) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var title, body string
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return title, body, err
	}
	solarSystem, err := n.eus.GetOrCreateSolarSystemESI(ctx, data.SolarSystemID)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("ESS main bank link in %s", solarSystem.Name)
	body = fmt.Sprintf(
		"%s has linked to the ESS main bank in %s.",
		makeEveEntityProfileLink(entities[data.CharID]),
		makeSolarSystemLink(solarSystem),
	)
	return title, body, nil
}

type expertSystemExpired struct {
	baseRenderer
}

func (n expertSystemExpired) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.ExpertSystemExpired
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	skillType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Expert system expired: %s", skillType.Name)
	body := fmt.Sprintf(
		"The expert system for **%s** has expired.",
		skillType.Name,
	)
	return title, body, nil
}

type expertSystemExpiryImminent struct {
	baseRenderer
}

func (n expertSystemExpiryImminent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	var data notification2.ExpertSystemExpiryImminent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	skillType, err := n.eus.GetOrCreateEntityESI(ctx, data.TypeID)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Expert system expiring soon: %s", skillType.Name)
	body := fmt.Sprintf(
		"The expert system for **%s** will expire in **%d** days.",
		skillType.Name,
		data.DaysUntilExpiry,
	)
	return title, body, nil
}

type allStructureInvulnerableMsg struct {
	baseRenderer
}

func (n allStructureInvulnerableMsg) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Alliance structures are invulnerable"
	body := "All alliance structures have been made invulnerable."
	return title, body, nil
}

type allStructVulnerableMsg struct {
	baseRenderer
}

func (n allStructVulnerableMsg) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Alliance structures are vulnerable"
	body := "Alliance structures have returned to their normal vulnerability state."
	return title, body, nil
}
