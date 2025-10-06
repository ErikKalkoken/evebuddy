// Package testutil contains factories for creating test objects in the repository
package testutil

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync/atomic"
	"time"

	"github.com/icrowley/fake"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

// EVE IDs
const (
	startIDAlliance      = 99_000_001
	startIDCelestials    = 40_000_001
	startIDCharacter     = 90_000_001
	startIDConstellation = 20_000_001
	startIDCorporation   = 98_000_001
	startIDFaction       = 500_001
	startIDInventoryType = 101
	startIDOther         = 10_001
	startIDRegion        = 10_000_001
	startIDSolarSystem   = 30_000_001
	startIDStation       = 60_000_001
	startIDStructure     = 1_000_000_000_001
)

type Factory struct {
	st   *storage.Storage
	dbRO *sql.DB
}

func NewFactory(st *storage.Storage, dbRO *sql.DB) Factory {
	f := Factory{st: st, dbRO: dbRO}
	return f
}

func (f Factory) RandomTime() time.Time {
	hours := time.Duration(rand.IntN(100_000))
	seconds := time.Duration(rand.IntN(3600))
	d := hours*time.Hour + seconds*time.Second
	return time.Now().Add(-d).UTC()
}

// CreateCharacter creates and returns a new character. Empty optional values are not filled.
func (f Factory) CreateCharacter(args ...storage.CreateCharacterParams) *app.Character {
	var arg storage.CreateCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		c := f.CreateEveCharacter()
		arg.ID = c.ID
	}
	ctx := context.Background()
	err := f.st.CreateCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	c, err := f.st.GetCharacter(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

func (f Factory) SetCharacterRoles(characterID int32, roles set.Set[app.Role]) {
	err := f.st.UpdateCharacterRoles(context.Background(), characterID, roles)
	if err != nil {
		panic(err)
	}
}

// CreateCharacterFull creates and returns a new character. Empty optionals are filled with random values.
func (f Factory) CreateCharacterFull(args ...storage.CreateCharacterParams) *app.Character {
	var arg storage.CreateCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		c := f.CreateEveCharacter()
		arg.ID = c.ID
	}
	if arg.AssetValue.IsEmpty() {
		arg.AssetValue = optional.New(rand.Float64() * 100_000_000_000)
	}
	if arg.HomeID.IsEmpty() {
		x := f.CreateEveLocationStructure()
		arg.HomeID = optional.New(x.ID)
	}
	if arg.LastCloneJumpAt.IsEmpty() {
		arg.LastCloneJumpAt = optional.New(time.Now().Add(-time.Duration(rand.IntN(10)) * time.Hour * 24).UTC())
	}
	if arg.LastLoginAt.IsEmpty() {
		arg.LastLoginAt = optional.New(time.Now().Add(-time.Duration(rand.IntN(10)) * time.Hour * 24).UTC())
	}
	if arg.LocationID.IsEmpty() {
		x := f.CreateEveLocationStructure()
		arg.LocationID = optional.New(x.ID)
	}
	if arg.ShipID.IsEmpty() {
		x := f.CreateEveType()
		arg.ShipID = optional.New(x.ID)
	}
	if arg.TotalSP.IsEmpty() {
		arg.TotalSP = optional.New(rand.IntN(100_000_000))
	}
	if arg.UnallocatedSP.IsEmpty() {
		arg.UnallocatedSP = optional.New(rand.IntN(10_000_000))
	}
	if arg.WalletBalance.IsEmpty() {
		arg.WalletBalance = optional.New(rand.Float64() * 100_000_000_000)
	}
	ctx := context.Background()
	err := f.st.CreateCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	c, err := f.st.GetCharacter(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

func (f Factory) CreateCharacterAttributes(args ...storage.UpdateOrCreateCharacterAttributesParams) *app.CharacterAttributes {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCharacterAttributesParams
	randomValue := func() int {
		return 20 + rand.IntN(5)
	}
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.Charisma == 0 {
		arg.Charisma = randomValue()
	}
	if arg.Intelligence == 0 {
		arg.Intelligence = randomValue()
	}
	if arg.Memory == 0 {
		arg.Memory = randomValue()
	}
	if arg.Perception == 0 {
		arg.Perception = randomValue()
	}
	if arg.Willpower == 0 {
		arg.Willpower = randomValue()
	}
	if err := f.st.UpdateOrCreateCharacterAttributes(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterAttributes(ctx, arg.CharacterID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterAsset(args ...storage.CreateCharacterAssetParams) *app.CharacterAsset {
	ctx := context.Background()
	var arg storage.CreateCharacterAssetParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.ItemID == 0 {
		arg.ItemID = f.calcNewIDWithCharacter("character_assets", "item_id", arg.CharacterID)
	}
	if arg.LocationFlag == app.FlagUndefined {
		arg.LocationFlag = app.FlagHangar
	}
	if arg.LocationID == 0 {
		x := f.CreateEveLocationStructure()
		arg.LocationID = x.ID
	}
	if arg.LocationType == app.TypeUndefined {
		arg.LocationType = app.TypeOther
	}
	if arg.IsSingleton && arg.Name == "" {
		arg.Name = fmt.Sprintf("Asset %d", arg.ItemID)
	}
	if arg.Quantity == 0 {
		if arg.IsSingleton {
			arg.Quantity = 1
		} else {
			arg.Quantity = rand.Int32N(10_000)
		}
	}
	if err := f.st.CreateCharacterAsset(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterAsset(ctx, arg.CharacterID, arg.ItemID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterContract(args ...storage.CreateCharacterContractParams) *app.CharacterContract {
	ctx := context.Background()
	var arg storage.CreateCharacterContractParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Availability == app.ContractAvailabilityUndefined {
		arg.Availability = app.ContractAvailabilityPublic
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.ContractID == 0 {
		arg.ContractID = int32(f.calcNewIDWithCharacter(
			"character_contracts",
			"contract_id",
			arg.CharacterID,
		))
	}
	if arg.DateIssued.IsZero() {
		arg.DateIssued = time.Now().UTC()
	}
	if arg.DateExpired.IsZero() {
		arg.DateExpired = arg.DateIssued.Add(time.Duration(rand.IntN(200)+12) * time.Hour)
	}
	if arg.IssuerID == 0 {
		c, err := f.st.GetCharacter(ctx, arg.CharacterID)
		if err != nil {
			panic(err)
		}
		arg2 := storage.CreateEveEntityParams{
			ID:       c.ID,
			Name:     c.EveCharacter.Name,
			Category: app.EveEntityCharacter,
		}
		_, err = f.st.GetOrCreateEveEntity(ctx, arg2)
		if err != nil {
			panic(err)
		}
		arg.IssuerID = c.ID
	}
	if arg.IssuerCorporationID == 0 {
		c, err := f.st.GetCharacter(ctx, arg.CharacterID)
		if err != nil {
			panic(err)
		}
		arg.IssuerCorporationID = c.EveCharacter.Corporation.ID
	}
	if arg.Status == app.ContractStatusUndefined {
		arg.Status = app.ContractStatusOutstanding
	}
	switch arg.Type {
	case app.ContractTypeCourier:
		if arg.EndLocationID == 0 {
			x := f.CreateEveLocationStructure()
			arg.EndLocationID = x.ID
		}
		if arg.StartLocationID == 0 {
			x := f.CreateEveLocationStructure()
			arg.StartLocationID = x.ID
		}
	case app.ContractTypeUndefined:
		arg.Type = app.ContractTypeItemExchange
	}
	_, err := f.st.CreateCharacterContract(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterContract(ctx, arg.CharacterID, arg.ContractID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterContractCourier(args ...storage.CreateCharacterContractParams) *app.CharacterContract {
	var arg storage.CreateCharacterContractParams
	if len(args) > 0 {
		arg = args[0]
	}
	arg.Type = app.ContractTypeCourier
	return f.CreateCharacterContract(arg)
}

func (f Factory) CreateCharacterContractBid(args ...storage.CreateCharacterContractBidParams) *app.CharacterContractBid {
	ctx := context.Background()
	var arg storage.CreateCharacterContractBidParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ContractID == 0 {
		c := f.CreateCharacterContract()
		arg.ContractID = c.ID
	}
	if arg.BidID == 0 {
		arg.BidID = int32(f.calcNewIDWithParam(
			"character_contract_bids",
			"bid_id",
			"contract_id",
			arg.ContractID,
		))
	}
	if arg.Amount == 0 {
		arg.Amount = rand.Float32() * 100_000_000
	}
	if arg.BidderID == 0 {
		x := f.CreateEveEntityCharacter()
		arg.BidderID = x.ID
	}
	if arg.DateBid.IsZero() {
		arg.DateBid = time.Now().UTC()
	}
	if err := f.st.CreateCharacterContractBid(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterContractBid(ctx, arg.ContractID, arg.BidID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterContractItem(args ...storage.CreateCharacterContractItemParams) *app.CharacterContractItem {
	ctx := context.Background()
	var arg storage.CreateCharacterContractItemParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ContractID == 0 {
		c := f.CreateCharacterContract()
		arg.ContractID = c.ID
	}
	if arg.RecordID == 0 {
		arg.RecordID = f.calcNewIDWithParam(
			"character_contract_items",
			"record_id",
			"contract_id",
			arg.ContractID,
		)
	}
	if arg.Quantity == 0 {
		arg.Quantity = int32(rand.IntN(10_000))
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	if err := f.st.CreateCharacterContractItem(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterContractItem(ctx, arg.ContractID, arg.RecordID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterImplant(args ...storage.CreateCharacterImplantParams) *app.CharacterImplant {
	ctx := context.Background()
	var arg storage.CreateCharacterImplantParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	err := f.st.CreateCharacterImplant(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterImplant(ctx, arg.CharacterID, arg.EveTypeID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterIndustryJob(args ...storage.UpdateOrCreateCharacterIndustryJobParams) *app.CharacterIndustryJob {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCharacterIndustryJobParams
	if len(args) > 0 {
		arg = args[0]
	}
	var character *app.Character
	if arg.CharacterID == 0 {
		character = f.CreateCharacterFull()
		arg.CharacterID = character.ID
	} else {
		x, err := f.st.GetCharacter(ctx, arg.CharacterID)
		if err != nil {
			panic(err)
		}
		character = x
	}
	if arg.StationID == 0 {
		x := f.CreateEveLocationStructure()
		arg.StationID = x.ID
	}
	if arg.ActivityID == 0 {
		activities := []app.IndustryActivity{
			app.Manufacturing,
			app.TimeEfficiencyResearch,
			app.MaterialEfficiencyResearch,
			app.Copying,
			app.Invention,
			app.Reactions2,
		}
		arg.ActivityID = int32(activities[rand.IntN(len(activities))])
	}
	if arg.BlueprintID == 0 {
		arg.BlueprintID = rand.Int64N(10_000_000)
	}
	if arg.BlueprintLocationID == 0 {
		arg.BlueprintLocationID = arg.StationID
	}
	if arg.BlueprintTypeID == 0 {
		x := f.CreateEveType()
		arg.BlueprintTypeID = x.ID
	}
	if arg.Duration == 0 {
		arg.Duration = rand.Int32N(1_000_000)
	}
	if arg.FacilityID == 0 {
		arg.FacilityID = arg.StationID
	}
	if arg.JobID == 0 {
		arg.JobID = int32(f.calcNewIDWithCharacter(
			"character_industry_jobs",
			"job_id",
			arg.CharacterID,
		))
	}
	if arg.InstallerID == 0 {
		f.st.GetOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:       character.ID,
			Name:     character.EveCharacter.Name,
			Category: app.EveEntityCharacter,
		})
		arg.InstallerID = character.ID
	}
	if arg.OutputLocationID == 0 {
		arg.OutputLocationID = arg.StationID
	}
	if arg.Runs == 0 {
		arg.Runs = rand.Int32N(50)
	}
	if arg.Status == 0 {
		items := []app.IndustryJobStatus{
			app.JobActive,
			app.JobCancelled,
			app.JobDelivered,
			app.JobPaused,
			app.JobReady,
			app.JobReverted,
		}
		arg.Status = items[rand.IntN(len(items))]
	}
	if arg.EndDate.IsZero() {
		arg.EndDate = time.Now().UTC().Add(time.Duration(rand.IntN(200)+12) * time.Hour)
	}
	if arg.StartDate.IsZero() {
		arg.StartDate = arg.EndDate.Add(-time.Duration(arg.Duration) * time.Second)
	}
	err := f.st.UpdateOrCreateCharacterIndustryJob(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterIndustryJob(ctx, arg.CharacterID, arg.JobID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterJumpClone(args ...storage.CreateCharacterJumpCloneParams) *app.CharacterJumpClone {
	ctx := context.Background()
	var arg storage.CreateCharacterJumpCloneParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.JumpCloneID == 0 {
		arg.JumpCloneID = f.calcNewIDWithCharacter(
			"character_jump_clones",
			"jump_clone_id",
			arg.CharacterID,
		)
	}
	if arg.LocationID == 0 {
		x := f.CreateEveLocationStructure()
		arg.LocationID = x.ID
	}
	if len(arg.Implants) == 0 {
		x := f.CreateEveType()
		arg.Implants = append(arg.Implants, x.ID)
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("JC-%d", arg.JumpCloneID)
	}
	err := f.st.CreateCharacterJumpClone(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterJumpClone(ctx, arg.CharacterID, int32(arg.JumpCloneID))
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterMail(args ...storage.CreateCharacterMailParams) *app.CharacterMail {
	var arg storage.CreateCharacterMailParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		c := f.CreateCharacterFull()
		arg.CharacterID = c.ID
	}
	if arg.FromID == 0 {
		from := f.CreateEveEntityCharacter()
		arg.FromID = from.ID
	}
	if arg.MailID == 0 {
		arg.MailID = int32(f.calcNewIDWithCharacter(
			"character_mails",
			"mail_id",
			arg.CharacterID,
		))
	}
	if arg.Body == "" {
		arg.Body = fake.Paragraph()
	}
	if arg.Subject == "" {
		arg.Subject = fake.Sentence()
	}
	if arg.Timestamp.IsZero() {
		arg.Timestamp = time.Now().UTC()
	}
	if len(arg.RecipientIDs) == 0 {
		e1 := f.CreateEveEntityCharacter()
		arg.RecipientIDs = []int32{e1.ID}
	}
	_, err := f.st.CreateCharacterMail(ctx, arg)
	if err != nil {
		panic(err)
	}
	mail, err := f.st.GetCharacterMail(ctx, arg.CharacterID, arg.MailID)
	if err != nil {
		panic(err)
	}
	return mail
}

func (f Factory) CreateCharacterMailLabel(args ...app.CharacterMailLabel) *app.CharacterMailLabel {
	ctx := context.Background()
	var arg storage.MailLabelParams
	if len(args) > 0 {
		l := args[0]
		arg = storage.MailLabelParams{
			CharacterID: l.CharacterID,
			Color:       l.Color,
			LabelID:     l.LabelID,
			Name:        l.Name,
			UnreadCount: l.UnreadCount,
		}
	}
	if arg.CharacterID == 0 {
		c := f.CreateCharacterFull()
		arg.CharacterID = c.ID
	}
	if arg.LabelID == 0 {
		l := int32(f.calcNewIDWithCharacter("character_mail_labels", "label_id", arg.CharacterID))
		arg.LabelID = max(l, 10) // generate "custom" mail label
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s %s", fake.Color(), fake.Language())
	}
	if arg.Color == "" {
		arg.Color = "#FFFFFF"
	}
	if arg.UnreadCount == 0 {
		arg.UnreadCount = int(rand.IntN(1000))
	}
	label, err := f.st.UpdateOrCreateCharacterMailLabel(ctx, arg)
	if err != nil {
		panic(err)
	}
	return label
}

func (f Factory) CreateCharacterMailList(characterID int32, args ...app.EveEntity) *app.EveEntity {
	var e app.EveEntity
	ctx := context.Background()
	if len(args) > 0 {
		e = args[0]
	}
	if characterID == 0 {
		c := f.CreateCharacterFull()
		characterID = c.ID
	}
	if e.ID == 0 {
		e = *f.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList})
	}
	if err := f.st.CreateCharacterMailList(ctx, characterID, e.ID); err != nil {
		panic(err)
	}
	return &e
}

func (f Factory) CreateCharacterPlanet(args ...storage.CreateCharacterPlanetParams) *app.CharacterPlanet {
	ctx := context.Background()
	var arg storage.CreateCharacterPlanetParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.EvePlanetID == 0 {
		x := f.CreateEvePlanet()
		arg.EvePlanetID = x.ID
	}
	if arg.UpgradeLevel == 0 {
		arg.UpgradeLevel = rand.IntN(5)
	}
	if arg.LastUpdate.IsZero() {
		arg.LastUpdate = time.Now().UTC()
	}
	_, err := f.st.CreateCharacterPlanet(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterPlanet(ctx, arg.CharacterID, arg.EvePlanetID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreatePlanetPin(args ...storage.CreatePlanetPinParams) *app.PlanetPin {
	ctx := context.Background()
	var arg storage.CreatePlanetPinParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterPlanetID == 0 {
		x := f.CreateCharacterPlanet()
		arg.CharacterPlanetID = x.ID
	}
	if arg.PinID == 0 {
		arg.PinID = f.calcNewID("planet_pins", "pin_id", 1)
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	if err := f.st.CreatePlanetPin(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetPlanetPin(ctx, arg.CharacterPlanetID, arg.PinID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreatePlanetPinExtractor(args ...storage.CreatePlanetPinParams) *app.PlanetPin {
	var arg storage.CreatePlanetPinParams
	if len(args) > 0 {
		arg = args[0]
	}
	eg := f.CreateEveGroup(storage.CreateEveGroupParams{ID: app.EveGroupExtractorControlUnits})
	et := f.CreateEveType(storage.CreateEveTypeParams{GroupID: eg.ID})
	arg.TypeID = et.ID
	return f.CreatePlanetPin(arg)
}

func (f Factory) CreateCharacterSkill(args ...storage.UpdateOrCreateCharacterSkillParams) *app.CharacterSkill {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCharacterSkillParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.TrainedSkillLevel == 0 {
		arg.TrainedSkillLevel = rand.IntN(5) + 1
	}
	if arg.ActiveSkillLevel == 0 {
		arg.TrainedSkillLevel = rand.IntN(arg.TrainedSkillLevel) + 1
	}
	if arg.SkillPointsInSkill == 0 {
		arg.SkillPointsInSkill = rand.IntN(1_000_000)
	}
	err := f.st.UpdateOrCreateCharacterSkill(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCharacterSkill(ctx, arg.CharacterID, arg.EveTypeID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterSkillqueueItem(args ...storage.SkillqueueItemParams) *app.CharacterSkillqueueItem {
	ctx := context.Background()
	var arg storage.SkillqueueItemParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.FinishedLevel == 0 {
		arg.FinishedLevel = rand.IntN(5) + 1
	}
	if arg.LevelEndSP == 0 {
		arg.LevelEndSP = rand.IntN(1_000_000)
	}
	if arg.QueuePosition == 0 {
		var maxPos sql.NullInt64
		q := "SELECT MAX(queue_position) FROM character_skillqueue_items WHERE character_id=?;"
		if err := f.dbRO.QueryRow(q, arg.CharacterID).Scan(&maxPos); err != nil {
			panic(err)
		}
		if maxPos.Valid {
			arg.QueuePosition = int(maxPos.Int64) + 1
		} else {
			arg.QueuePosition = int(maxPos.Int64) + 1
		}
	}
	if arg.StartDate.IsZero() {
		var v sql.NullString
		q2 := "SELECT MAX(finish_date) FROM character_skillqueue_items WHERE character_id=?;"
		if err := f.dbRO.QueryRow(q2, arg.CharacterID).Scan(&v); err != nil {
			panic(err)
		}
		if !v.Valid {
			arg.StartDate = time.Now().UTC()
		} else {
			maxFinishDate, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", v.String)
			if err != nil {
				panic(err)
			}
			arg.StartDate = maxFinishDate
		}
	}
	if arg.FinishDate.IsZero() {
		hours := rand.IntN(90)*24 + 3
		arg.FinishDate = arg.StartDate.Add(time.Hour * time.Duration(hours))
	}
	err := f.st.CreateCharacterSkillqueueItem(ctx, arg)
	if err != nil {
		panic(err)
	}
	i, err := f.st.GetCharacterSkillqueueItem(ctx, arg.CharacterID, arg.QueuePosition)
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateCharacterTag(names ...string) *app.CharacterTag {
	var name string
	if len(names) > 0 {
		name = names[0]
	}
	ctx := context.Background()
	if name == "" {
		name = fmt.Sprintf("%s #%d", fake.Color(), rand.IntN(1000))
	}
	r, err := f.st.CreateTag(ctx, name)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) AddCharacterToTag(tag *app.CharacterTag, character *app.Character) {
	err := f.st.CreateCharactersCharacterTag(context.Background(), storage.CreateCharacterTagParams{
		CharacterID: character.ID,
		TagID:       tag.ID,
	})
	if err != nil {
		panic(err)
	}
}

func (f Factory) CreateCharacterToken(args ...storage.UpdateOrCreateCharacterTokenParams) *app.CharacterToken {
	var arg storage.UpdateOrCreateCharacterTokenParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.AccessToken == "" {
		arg.AccessToken = fmt.Sprintf("GeneratedAccessToken#%d", rand.IntN(1000000))
	}
	if arg.RefreshToken == "" {
		arg.RefreshToken = fmt.Sprintf("GeneratedRefreshToken#%d", rand.IntN(1000000))
	}
	if arg.ExpiresAt.IsZero() {
		arg.ExpiresAt = time.Now().Add(time.Minute * 20).UTC()
	}
	if arg.TokenType == "" {
		arg.TokenType = "Bearer"
	}
	if arg.Scopes.Size() == 0 {
		arg.Scopes = app.Scopes()
	}
	if arg.CharacterID == 0 {
		c := f.CreateCharacterFull()
		arg.CharacterID = c.ID
	}
	err := f.st.UpdateOrCreateCharacterToken(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCharacterToken(ctx, arg.CharacterID)
	if err != nil {
		panic(err)
	}
	return x
}

type CharacterSectionStatusParams struct {
	CharacterID  int32
	CompletedAt  time.Time
	Data         any
	ErrorMessage string
	Section      app.CharacterSection
	StartedAt    time.Time
	UpdatedAt    time.Time
}

func (f Factory) CreateCharacterSectionStatus(args ...CharacterSectionStatusParams) *app.CharacterSectionStatus {
	ctx := context.Background()
	var arg CharacterSectionStatusParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		c := f.CreateCharacterFull()
		arg.CharacterID = c.ID
	}
	if arg.Section == "" {
		panic("must define a section in test factory")
	}
	if arg.Data == "" {
		arg.Data = fmt.Sprintf("content-hash-%d-%s-%s", arg.CharacterID, arg.Section, time.Now())
	}
	if arg.CompletedAt.IsZero() {
		arg.CompletedAt = time.Now().UTC()
	}
	if arg.UpdatedAt.IsZero() {
		arg.UpdatedAt = time.Now().UTC()
	}
	if arg.StartedAt.IsZero() {
		arg.StartedAt = time.Now().Add(-1 * time.Duration(rand.IntN(60)) * time.Second).UTC()
	}
	hash, err := calcContentHash(arg.Data)
	if err != nil {
		panic(err)
	}
	t := storage.NewNullTimeFromTime(arg.CompletedAt)
	o, err := f.st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
		CharacterID:  arg.CharacterID,
		Section:      arg.Section,
		ErrorMessage: &arg.ErrorMessage,
		CompletedAt:  &t,
		ContentHash:  &hash,
		UpdatedAt:    &arg.UpdatedAt,
	})
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCharacterWalletJournalEntry(args ...storage.CreateCharacterWalletJournalEntryParams) *app.CharacterWalletJournalEntry {
	ctx := context.Background()
	var arg storage.CreateCharacterWalletJournalEntryParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.RefID == 0 {
		arg.RefID = int64(f.calcNewIDWithCharacter("character_wallet_journal_entries", "id", arg.CharacterID))
	}
	if arg.Amount == 0 {
		var f float64
		if rand.Float32() > 0.5 {
			f = 1
		} else {
			f = -1
		}
		arg.Amount = rand.Float64() * 10_000_000_000 * f
	}
	if arg.Balance == 0 {
		arg.Balance = rand.Float64() * 100_000_000_000
	}
	if arg.Date.IsZero() {
		arg.Date = time.Now().UTC()
	}
	if arg.Description == "" {
		arg.Description = fake.Sentence()
	}
	if arg.Reason == "" {
		arg.Reason = fake.Sentence()
	}
	if arg.RefType == "" {
		arg.RefType = "player_donation"
	}
	if arg.Tax == 0 {
		arg.Tax = rand.Float64()
	}
	if arg.FirstPartyID == 0 {
		e := f.CreateEveEntityCharacter()
		arg.FirstPartyID = e.ID
	}
	if arg.SecondPartyID == 0 {
		e := f.CreateEveEntityCharacter()
		arg.SecondPartyID = e.ID
	}
	if arg.TaxReceiverID == 0 {
		e := f.CreateEveEntityCorporation()
		arg.TaxReceiverID = e.ID
	}
	err := f.st.CreateCharacterWalletJournalEntry(ctx, arg)
	if err != nil {
		panic(fmt.Sprintf("%s|%+v", err, arg))
	}
	i, err := f.st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
		CharacterID: arg.CharacterID,
		RefID:       arg.RefID,
	})
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateCharacterWalletTransaction(args ...storage.CreateCharacterWalletTransactionParams) *app.CharacterWalletTransaction {
	ctx := context.Background()
	var arg storage.CreateCharacterWalletTransactionParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ClientID == 0 {
		x := f.CreateEveEntityCharacter()
		arg.ClientID = x.ID
	}
	if arg.Date.IsZero() {
		arg.Date = time.Now().UTC()
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.LocationID == 0 {
		x := f.CreateEveLocationStructure()
		arg.LocationID = x.ID
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.TransactionID == 0 {
		arg.TransactionID = f.calcNewIDWithCharacter(
			"character_wallet_transactions",
			"transaction_id",
			arg.CharacterID,
		)
	}
	if arg.UnitPrice == 0 {
		arg.UnitPrice = rand.Float64() * 100_000_000
	}
	if arg.Quantity == 0 {
		arg.Quantity = rand.Int32N(100_000)
	}
	if arg.JournalRefID == 0 {
		x := f.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{
			CharacterID: arg.CharacterID,
		})
		arg.JournalRefID = x.ID
	}
	err := f.st.CreateCharacterWalletTransaction(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCharacterWalletTransaction(ctx, storage.GetCharacterWalletTransactionParams{
		CharacterID:   arg.CharacterID,
		TransactionID: arg.TransactionID,
	})
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCharacterMarketOrder(args ...storage.UpdateOrCreateCharacterMarketOrderParams) *app.CharacterMarketOrder {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCharacterMarketOrderParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacter()
		arg.CharacterID = x.ID
	}
	if arg.Duration == 0 {
		arg.Duration = rand.IntN(30) + 1
	}
	if arg.Issued.IsZero() {
		arg.Issued = time.Now().UTC()
	}
	if arg.OrderID == 0 {
		arg.OrderID = f.calcNewIDWithCharacter(
			"character_market_orders",
			"order_id",
			arg.CharacterID,
		)
	}
	if arg.OwnerID == 0 {
		x, err := f.st.GetOrCreateEveEntity(ctx, storage.CreateEveEntityParams{
			ID:       arg.CharacterID,
			Category: app.EveEntityCharacter,
			Name:     "PLACEHOLDER",
		})
		if err != nil {
			panic(err)
		}
		arg.OwnerID = x.ID
	}
	if arg.Price == 0 {
		arg.Price = rand.Float64()*100_000_000 + 1
	}
	if arg.Range == "" {
		arg.Range = "station"
	}
	if arg.RegionID == 0 {
		x := f.CreateEveRegion()
		arg.RegionID = x.ID
	}
	if arg.LocationID == 0 {
		c := f.CreateEveConstellation(storage.CreateEveConstellationParams{
			RegionID: arg.RegionID,
		})
		s := f.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
			ConstellationID: c.ID,
		})
		x := f.CreateEveLocationStation(storage.UpdateOrCreateLocationParams{
			SolarSystemID: optional.New(s.ID),
		})
		arg.LocationID = x.ID
	}
	if arg.State == app.OrderUndefined {
		arg.State = app.OrderOpen
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	if arg.VolumeTotal == 0 {
		arg.VolumeTotal = rand.IntN(100_000) + 1
	}
	if arg.VolumeRemains == 0 {
		arg.VolumeTotal = max(rand.IntN(arg.VolumeTotal), 1)
	}
	err := f.st.UpdateOrCreateCharacterMarketOrder(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCharacterMarketOrder(ctx, arg.CharacterID, arg.OrderID)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCharacterNotification(args ...storage.CreateCharacterNotificationParams) *app.CharacterNotification {
	ctx := context.Background()
	var arg storage.CreateCharacterNotificationParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CharacterID == 0 {
		x := f.CreateCharacterFull()
		arg.CharacterID = x.ID
	}
	if arg.NotificationID == 0 {
		arg.NotificationID = f.calcNewIDWithCharacter(
			"character_notifications",
			"notification_id",
			arg.CharacterID,
		)
	}
	if arg.SenderID == 0 {
		x := f.CreateEveEntityCorporation()
		arg.SenderID = x.ID
	}
	if arg.RecipientID.IsEmpty() {
		x := f.CreateEveEntityCorporation()
		arg.RecipientID.Set(x.ID)
	}
	if arg.Type == "" {
		arg.Type = "CorpBecameWarEligible" // Type without text
	}
	if arg.Timestamp.IsZero() {
		arg.Timestamp = time.Now().UTC()
	}
	err := f.st.CreateCharacterNotification(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCharacterNotification(ctx, arg.CharacterID, arg.NotificationID)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCorporation(corporationID ...int32) *app.Corporation {
	var id int32
	if len(corporationID) == 0 {
		ec := f.CreateEveCorporation()
		id = ec.ID
	} else {
		id = corporationID[0]
		_, err := f.st.GetEveCorporation(context.Background(), id)
		if errors.Is(err, app.ErrNotFound) {
			f.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
				ID: corporationID[0],
			})
		} else if err != nil {
			panic(err)
		}
	}
	err := f.st.CreateCorporation(context.Background(), id)
	if err != nil {
		panic(err)
	}
	c, err := f.st.GetCorporation(context.Background(), id)
	if err != nil {
		panic(err)
	}
	return c
}

func (f Factory) CreateCorporationContract(args ...storage.CreateCorporationContractParams) *app.CorporationContract {
	ctx := context.Background()
	var arg storage.CreateCorporationContractParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Availability == app.ContractAvailabilityUndefined {
		arg.Availability = app.ContractAvailabilityPublic
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.ContractID == 0 {
		arg.ContractID = int32(f.calcNewIDWithCorporation(
			"corporation_contracts",
			"contract_id",
			arg.CorporationID,
		))
	}
	if arg.DateIssued.IsZero() {
		arg.DateIssued = time.Now().UTC()
	}
	if arg.DateExpired.IsZero() {
		arg.DateExpired = arg.DateIssued.Add(time.Duration(rand.IntN(200)+12) * time.Hour)
	}
	if arg.IssuerID == 0 {
		c, err := f.st.GetCorporation(ctx, arg.CorporationID)
		if err != nil {
			panic(err)
		}
		arg2 := storage.CreateEveEntityParams{
			ID:       c.ID,
			Name:     c.EveCorporation.Name,
			Category: app.EveEntityCorporation,
		}
		_, err = f.st.GetOrCreateEveEntity(ctx, arg2)
		if err != nil {
			panic(err)
		}
		arg.IssuerID = c.ID
	}
	if arg.IssuerCorporationID == 0 {
		c, err := f.st.GetCorporation(ctx, arg.CorporationID)
		if err != nil {
			panic(err)
		}
		arg.IssuerCorporationID = c.EveCorporation.ID
	}
	if arg.Status == app.ContractStatusUndefined {
		arg.Status = app.ContractStatusOutstanding
	}
	switch arg.Type {
	case app.ContractTypeCourier:
		if arg.EndLocationID == 0 {
			x := f.CreateEveLocationStructure()
			arg.EndLocationID = x.ID
		}
		if arg.StartLocationID == 0 {
			x := f.CreateEveLocationStructure()
			arg.StartLocationID = x.ID
		}
	case app.ContractTypeUndefined:
		arg.Type = app.ContractTypeItemExchange
	}
	_, err := f.st.CreateCorporationContract(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCorporationContract(ctx, arg.CorporationID, arg.ContractID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCorporationContractCourier(args ...storage.CreateCorporationContractParams) *app.CorporationContract {
	var arg storage.CreateCorporationContractParams
	if len(args) > 0 {
		arg = args[0]
	}
	arg.Type = app.ContractTypeCourier
	return f.CreateCorporationContract(arg)
}

func (f Factory) CreateCorporationContractBid(args ...storage.CreateCorporationContractBidParams) *app.CorporationContractBid {
	ctx := context.Background()
	var arg storage.CreateCorporationContractBidParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ContractID == 0 {
		c := f.CreateCorporationContract()
		arg.ContractID = c.ID
	}
	if arg.BidID == 0 {
		arg.BidID = int32(f.calcNewIDWithParam(
			"corporation_contract_bids",
			"bid_id",
			"contract_id",
			arg.ContractID,
		))
	}
	if arg.Amount == 0 {
		arg.Amount = rand.Float32() * 100_000_000
	}
	if arg.BidderID == 0 {
		x := f.CreateEveEntityCorporation()
		arg.BidderID = x.ID
	}
	if arg.DateBid.IsZero() {
		arg.DateBid = time.Now().UTC()
	}
	if err := f.st.CreateCorporationContractBid(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCorporationContractBid(ctx, arg.ContractID, arg.BidID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCorporationContractItem(args ...storage.CreateCorporationContractItemParams) *app.CorporationContractItem {
	ctx := context.Background()
	var arg storage.CreateCorporationContractItemParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ContractID == 0 {
		c := f.CreateCorporationContract()
		arg.ContractID = c.ID
	}
	if arg.RecordID == 0 {
		arg.RecordID = f.calcNewIDWithParam(
			"corporation_contract_items",
			"record_id",
			"contract_id",
			arg.ContractID,
		)
	}
	if arg.Quantity == 0 {
		arg.Quantity = int32(rand.IntN(10_000))
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	if err := f.st.CreateCorporationContractItem(ctx, arg); err != nil {
		panic(err)
	}
	o, err := f.st.GetCorporationContractItem(ctx, arg.ContractID, arg.RecordID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCorporationHangarName(args ...storage.UpdateOrCreateCorporationHangarNameParams) *app.CorporationHangarName {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCorporationHangarNameParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.DivisionID == 0 {
		arg.DivisionID = 1
	}
	if arg.Name == "" {
		arg.Name = fake.Color()
	}
	err := f.st.UpdateOrCreateCorporationHangarName(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCorporationHangarName(ctx, storage.CorporationDivision{
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
	})
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCorporationMember(args ...storage.CorporationMemberParams) *app.CorporationMember {
	ctx := context.Background()
	var arg storage.CorporationMemberParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.CharacterID == 0 {
		x := f.CreateEveEntityCharacter()
		arg.CharacterID = x.ID
	}
	err := f.st.CreateCorporationMember(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCorporationMember(ctx, arg)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCorporationIndustryJob(args ...storage.UpdateOrCreateCorporationIndustryJobParams) *app.CorporationIndustryJob {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCorporationIndustryJobParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.ActivityID == 0 {
		activities := []app.IndustryActivity{
			app.Manufacturing,
			app.TimeEfficiencyResearch,
			app.MaterialEfficiencyResearch,
			app.Copying,
			app.Invention,
			app.Reactions2,
		}
		arg.ActivityID = int32(activities[rand.IntN(len(activities))])
	}
	if arg.BlueprintID == 0 {
		arg.BlueprintID = rand.Int64N(10_000_000)
	}
	if arg.BlueprintLocationID == 0 {
		arg.BlueprintLocationID = rand.Int64N(10_000_000_000)
	}
	if arg.BlueprintTypeID == 0 {
		x := f.CreateEveType()
		arg.BlueprintTypeID = x.ID
	}
	if arg.Duration == 0 {
		arg.Duration = rand.Int32N(10_000)
	}
	if arg.FacilityID == 0 {
		arg.FacilityID = rand.Int64N(10_000_000_000)
	}
	if arg.JobID == 0 {
		arg.JobID = int32(f.calcNewIDWithCorporation(
			"corporation_industry_jobs",
			"job_id",
			arg.CorporationID,
		))
	}
	if arg.InstallerID == 0 {
		x := f.CreateEveEntityCharacter()
		arg.InstallerID = x.ID
	}
	if arg.OutputLocationID == 0 {
		arg.OutputLocationID = rand.Int64N(10_000_000_000)
	}
	if arg.Runs == 0 {
		arg.Runs = rand.Int32N(50)
	}
	if arg.LocationID == 0 {
		x := f.CreateEveLocationStructure()
		arg.LocationID = x.ID
	}
	if arg.Status == 0 {
		items := []app.IndustryJobStatus{
			app.JobActive,
			app.JobCancelled,
			app.JobDelivered,
			app.JobPaused,
			app.JobReady,
			app.JobReverted,
		}
		arg.Status = items[rand.IntN(len(items))]
	}
	now := time.Now().UTC()
	if arg.StartDate.IsZero() {
		arg.StartDate = now.Add(-time.Duration(rand.IntN(200)+12) * time.Hour)
	}
	if arg.EndDate.IsZero() {
		arg.EndDate = now.Add(time.Duration(rand.IntN(200)+12) * time.Hour)
	}
	err := f.st.UpdateOrCreateCorporationIndustryJob(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCorporationIndustryJob(ctx, arg.CorporationID, arg.JobID)
	if err != nil {
		panic(err)
	}
	return o
}

type CorporationSectionStatusParams struct {
	Comment       string
	CorporationID int32
	Section       app.CorporationSection
	ErrorMessage  string
	CompletedAt   time.Time
	StartedAt     time.Time
	Data          any
}

func (f Factory) CreateCorporationSectionStatus(args ...CorporationSectionStatusParams) *app.CorporationSectionStatus {
	ctx := context.Background()
	var arg CorporationSectionStatusParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		c := f.CreateCorporation()
		arg.CorporationID = c.ID
	}
	if arg.Section == "" {
		panic("must define a section in test factory")
	}
	if arg.Data == "" {
		arg.Data = fmt.Sprintf("content-hash-%d-%s-%s", arg.CorporationID, arg.Section, time.Now())
	}
	if arg.CompletedAt.IsZero() {
		arg.CompletedAt = time.Now().UTC()
	}
	if arg.StartedAt.IsZero() {
		arg.StartedAt = time.Now().Add(-1 * time.Duration(rand.IntN(60)) * time.Second).UTC()
	}
	hash, err := calcContentHash(arg.Data)
	if err != nil {
		panic(err)
	}
	t := storage.NewNullTimeFromTime(arg.CompletedAt)
	arg2 := storage.UpdateOrCreateCorporationSectionStatusParams{
		Comment:       &arg.Comment,
		CorporationID: arg.CorporationID,
		Section:       arg.Section,
		ErrorMessage:  &arg.ErrorMessage,
		CompletedAt:   &t,
		ContentHash:   &hash,
	}
	o, err := f.st.UpdateOrCreateCorporationSectionStatus(ctx, arg2)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCorporationStructure(args ...storage.UpdateOrCreateCorporationStructureParams) *app.CorporationStructure {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCorporationStructureParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.StructureID == 0 {
		arg.StructureID = f.calcNewIDWithCorporation(
			"corporation_structures",
			"structure_id",
			arg.CorporationID,
		)
	}
	if arg.State == app.StructureStateUndefined {
		arg.State = app.StructureStateShieldVulnerable
	}
	if arg.ProfileID == 0 {
		arg.ProfileID = rand.Int64N(10_000_000)
	}
	if arg.SystemID == 0 {
		x := f.CreateEveSolarSystem()
		arg.SystemID = x.ID
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	if arg.Name == "" {
		arg.Name = fake.City()
	}
	if arg.ReinforceHour.IsEmpty() {
		arg.ReinforceHour.Set(rand.Int64N(24))
	}
	if arg.FuelExpires.IsEmpty() {
		arg.FuelExpires.Set(time.Now().UTC().Add(time.Duration(rand.IntN(1000) * int(time.Hour))))
	}
	err := f.st.UpdateOrCreateCorporationStructure(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetCorporationStructure(ctx, arg.CorporationID, arg.StructureID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateCorporationWalletBalance(args ...storage.UpdateOrCreateCorporationWalletBalanceParams) *app.CorporationWalletBalance {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCorporationWalletBalanceParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.DivisionID == 0 {
		arg.DivisionID = 1
	}
	if arg.Balance == 0 {
		arg.Balance = rand.Float64()*100_000_000_000 + rand.Float64()
	}
	err := f.st.UpdateOrCreateCorporationWalletBalance(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCorporationWalletBalance(ctx, storage.CorporationDivision{
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
	})
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCorporationWalletName(args ...storage.UpdateOrCreateCorporationWalletNameParams) *app.CorporationWalletName {
	ctx := context.Background()
	var arg storage.UpdateOrCreateCorporationWalletNameParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.DivisionID == 0 {
		arg.DivisionID = 1
	}
	if arg.Name == "" {
		arg.Name = fake.Color()
	}
	err := f.st.UpdateOrCreateCorporationWalletName(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCorporationWalletName(ctx, storage.CorporationDivision{
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
	})
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateCorporationWalletJournalEntry(args ...storage.CreateCorporationWalletJournalEntryParams) *app.CorporationWalletJournalEntry {
	ctx := context.Background()
	var arg storage.CreateCorporationWalletJournalEntryParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.DivisionID == 0 {
		arg.DivisionID = 1
	}
	if arg.RefID == 0 {
		arg.RefID = int64(f.calcNewIDWithParams("corporation_wallet_journal_entries", "id", map[string]any{
			"corporation_id": arg.CorporationID,
			"division_id":    arg.DivisionID,
		}))
	}
	if arg.Amount == 0 {
		var f float64
		if rand.Float32() > 0.5 {
			f = 1
		} else {
			f = -1
		}
		arg.Amount = rand.Float64() * 10_000_000_000 * f
	}
	if arg.Balance == 0 {
		arg.Balance = rand.Float64() * 100_000_000_000
	}
	if arg.Date.IsZero() {
		arg.Date = time.Now().UTC()
	}
	if arg.Description == "" {
		arg.Description = fake.Sentence()
	}
	if arg.Reason == "" {
		arg.Reason = fake.Sentence()
	}
	if arg.RefType == "" {
		arg.RefType = "player_donation"
	}
	if arg.Tax == 0 {
		arg.Tax = rand.Float64()
	}
	if arg.FirstPartyID == 0 {
		e := f.CreateEveEntityCorporation()
		arg.FirstPartyID = e.ID
	}
	if arg.SecondPartyID == 0 {
		e := f.CreateEveEntityCorporation()
		arg.SecondPartyID = e.ID
	}
	if arg.TaxReceiverID == 0 {
		e := f.CreateEveEntityCorporation()
		arg.TaxReceiverID = e.ID
	}
	err := f.st.CreateCorporationWalletJournalEntry(ctx, arg)
	if err != nil {
		panic(fmt.Sprintf("%s|%+v", err, arg))
	}
	i, err := f.st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
		RefID:         arg.RefID,
	})
	if err != nil {
		panic(err)
	}
	return i
}

func (f Factory) CreateCorporationWalletTransaction(args ...storage.CreateCorporationWalletTransactionParams) *app.CorporationWalletTransaction {
	ctx := context.Background()
	var arg storage.CreateCorporationWalletTransactionParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationID == 0 {
		x := f.CreateCorporation()
		arg.CorporationID = x.ID
	}
	if arg.DivisionID == 0 {
		arg.DivisionID = 1
	}
	if arg.TransactionID == 0 {
		arg.TransactionID = f.calcNewIDWithParams(
			"corporation_wallet_transactions",
			"transaction_id",
			map[string]any{
				"corporation_id": arg.CorporationID,
				"division_id":    arg.DivisionID,
			},
		)
	}
	if arg.ClientID == 0 {
		x := f.CreateEveEntityCorporation()
		arg.ClientID = x.ID
	}
	if arg.Date.IsZero() {
		arg.Date = time.Now().UTC()
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.LocationID == 0 {
		x := f.CreateEveLocationStructure()
		arg.LocationID = x.ID
	}
	if arg.UnitPrice == 0 {
		arg.UnitPrice = rand.Float64() * 100_000_000
	}
	if arg.Quantity == 0 {
		arg.Quantity = rand.Int32N(100_000)
	}
	if arg.JournalRefID == 0 {
		x := f.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: arg.CorporationID,
		})
		arg.JournalRefID = x.ID
	}
	err := f.st.CreateCorporationWalletTransaction(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetCorporationWalletTransaction(ctx, storage.GetCorporationWalletTransactionParams{
		CorporationID: arg.CorporationID,
		DivisionID:    arg.DivisionID,
		TransactionID: arg.TransactionID,
	})
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateEveCharacter(args ...storage.CreateEveCharacterParams) *app.EveCharacter {
	ctx := context.Background()
	var arg storage.CreateEveCharacterParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_characters", "id", startIDCharacter))
	}
	if arg.Name == "" {
		arg.Name = fake.FullName()
	}
	x := f.GetOrCreateEveEntity(app.EveEntity{
		ID:       arg.CorporationID,
		Category: app.EveEntityCorporation,
	})
	arg.CorporationID = x.ID
	if arg.Birthday.IsZero() {
		arg.Birthday = time.Now().UTC().Add(-time.Duration(rand.IntN(10000)) * time.Hour * 24)
	}
	if arg.Description == "" {
		arg.Description = fake.Paragraphs()
	}
	if arg.Gender == "" {
		arg.Gender = "male"
	}
	if arg.RaceID == 0 {
		r := f.CreateEveRace()
		arg.RaceID = r.ID
	}
	if arg.Title == "" {
		arg.Title = fake.JobTitle()
	}
	err := f.st.UpdateOrCreateEveCharacter(ctx, arg)
	if err != nil {
		panic(err)
	}
	c, err := f.st.GetEveCharacter(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

func (f Factory) CreateEveCorporation(args ...storage.UpdateOrCreateEveCorporationParams) *app.EveCorporation {
	var arg storage.UpdateOrCreateEveCorporationParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_corporations", "id", startIDCorporation))
	}
	if arg.Name == "" {
		arg.Name = fake.Company()
	}
	if arg.CeoID.IsEmpty() {
		c := f.CreateEveEntityCharacter()
		arg.CeoID.Set(c.ID)
	}
	if arg.CreatorID.IsEmpty() {
		c := f.CreateEveEntityCharacter()
		arg.CreatorID.Set(c.ID)
	}
	if arg.DateFounded.IsEmpty() {
		arg.DateFounded = optional.New(time.Now().Add(-100 * time.Hour).UTC())
	}
	if arg.Description == "" {
		arg.Description = fake.Paragraphs()
	}
	if arg.MemberCount == 0 {
		arg.MemberCount = rand.Int32N(1000 + 1)
	}
	err := f.st.UpdateOrCreateEveCorporation(context.Background(), arg)
	if err != nil {
		panic(err)
	}
	c, err := f.st.GetEveCorporation(context.Background(), arg.ID)
	if err != nil {
		panic(err)
	}
	return c
}

type GeneralSectionStatusParams struct {
	Section      app.GeneralSection
	ErrorMessage string
	CompletedAt  time.Time
	StartedAt    time.Time
	Data         any
}

func (f Factory) CreateGeneralSectionStatus(args ...GeneralSectionStatusParams) *app.GeneralSectionStatus {
	ctx := context.Background()
	var arg GeneralSectionStatusParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Section == "" {
		panic("must define a section in test factory")
	}
	if arg.Data == "" {
		arg.Data = fmt.Sprintf("content-hash-%s-%s", arg.Section, time.Now())
	}
	if arg.CompletedAt.IsZero() {
		arg.CompletedAt = time.Now().UTC()
	}
	if arg.StartedAt.IsZero() {
		arg.StartedAt = time.Now().Add(-1 * time.Duration(rand.IntN(60)) * time.Second).UTC()
	}
	hash, err := calcContentHash(arg.Data)
	if err != nil {
		panic(err)
	}
	t := storage.NewNullTimeFromTime(arg.CompletedAt)
	arg2 := storage.UpdateOrCreateGeneralSectionStatusParams{
		Section:     arg.Section,
		Error:       &arg.ErrorMessage,
		CompletedAt: &t,
		ContentHash: &hash,
	}
	o, err := f.st.UpdateOrCreateGeneralSectionStatus(ctx, arg2)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) GetOrCreateEveEntity(args ...app.EveEntity) *app.EveEntity {
	var arg app.EveEntity
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID != 0 {
		o, err := f.st.GetEveEntity(context.Background(), arg.ID)
		if errors.Is(err, app.ErrNotFound) {
			// continue
		} else if err != nil {
			panic(err)
		} else {
			return o
		}
	}
	return f.CreateEveEntity(args...)
}

// TODO: Refactor to use storage.CreateEveEntityParams

func (f Factory) CreateEveEntity(args ...app.EveEntity) *app.EveEntity {
	ctx := context.Background()
	var arg app.EveEntity
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Category == app.EveEntityUndefined {
		arg.Category = app.EveEntityCharacter
	}
	if arg.ID == 0 {
		var start int64
		m := map[app.EveEntityCategory]int64{
			app.EveEntityAlliance:      startIDAlliance,
			app.EveEntityCharacter:     startIDCharacter,
			app.EveEntityCorporation:   startIDCorporation,
			app.EveEntityFaction:       startIDFaction,
			app.EveEntityInventoryType: startIDInventoryType,
			app.EveEntitySolarSystem:   startIDSolarSystem,
			app.EveEntityStation:       startIDStation,
		}
		start, ok := m[arg.Category]
		if !ok {
			start = startIDOther
		}
		arg.ID = int32(f.calcNewID("eve_entities", "id", start))
	}
	if arg.Name == "" {
		switch arg.Category {
		case app.EveEntityCharacter:
			arg.Name = fake.FullName()
		case app.EveEntityCorporation:
			arg.Name = fake.Company()
		case app.EveEntityAlliance:
			arg.Name = fake.Company()
		case app.EveEntityFaction:
			arg.Name = fake.JobTitle()
		case app.EveEntityMailList:
			arg.Name = fmt.Sprintf("%s %s", fake.Color(), fake.Industry())
		default:
			arg.Name = fmt.Sprintf("%s #%d", arg.Category, arg.ID)
		}
	}
	e, err := f.st.CreateEveEntity(ctx, storage.CreateEveEntityParams{ID: arg.ID, Name: arg.Name, Category: arg.Category})
	if err != nil {
		panic(fmt.Sprintf("create EveEntity %v: %s", arg, err))
	}
	return e
}

func (f Factory) CreateEveEntityAlliance(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntityAlliance)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCharacter(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntityCharacter)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityCorporation(args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, app.EveEntityCorporation)
	return f.CreateEveEntity(args2...)
}

func (f Factory) CreateEveEntityWithCategory(c app.EveEntityCategory, args ...app.EveEntity) *app.EveEntity {
	args2 := eveEntityWithCategory(args, c)
	return f.CreateEveEntity(args2...)
}

func eveEntityWithCategory(args []app.EveEntity, category app.EveEntityCategory) []app.EveEntity {
	var e app.EveEntity
	if len(args) > 0 {
		e = args[0]
	}
	e.Category = category
	args2 := []app.EveEntity{e}
	return args2
}

func (f Factory) CreateEveCategory(args ...storage.CreateEveCategoryParams) *app.EveCategory {
	var arg storage.CreateEveCategoryParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_categories", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fake.Industry()
	}
	o, err := f.st.GetOrCreateEveCategory(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveGroup(args ...storage.CreateEveGroupParams) *app.EveGroup {
	var arg storage.CreateEveGroupParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_groups", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fake.Brand()
	}
	x := f.CreateEveCategory(storage.CreateEveCategoryParams{ID: arg.CategoryID})
	arg.CategoryID = x.ID
	o, err := f.st.GetOrCreateEveGroup(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveShipSkill(args ...storage.CreateShipSkillParams) *app.EveShipSkill {
	var arg storage.CreateShipSkillParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	ship := f.CreateEveCategory(storage.CreateEveCategoryParams{
		ID:          app.EveCategoryShip,
		IsPublished: true,
		Name:        "Ship",
	})
	carrier := f.CreateEveGroup(storage.CreateEveGroupParams{
		CategoryID:  ship.ID,
		ID:          app.EveGroupCarrier,
		IsPublished: true,
		Name:        "Carrier",
	})
	shipType := f.CreateEveType(storage.CreateEveTypeParams{
		GroupID:     carrier.ID,
		ID:          arg.ShipTypeID,
		IsPublished: true,
	})
	arg.ShipTypeID = shipType.ID
	skill := f.CreateEveCategory(storage.CreateEveCategoryParams{
		ID:          app.EveCategorySkill,
		IsPublished: true,
		Name:        "Skill",
	})
	skillGroup := f.CreateEveGroup(storage.CreateEveGroupParams{
		CategoryID:  skill.ID,
		IsPublished: true,
	})
	skillType := f.CreateEveType(storage.CreateEveTypeParams{
		GroupID:     skillGroup.ID,
		ID:          arg.SkillTypeID,
		IsPublished: true,
	})
	arg.SkillTypeID = skillType.ID
	if arg.Rank == 0 {
		arg.Rank = 1
	}
	if arg.SkillLevel == 0 {
		arg.SkillLevel = 1
	}
	err := f.st.CreateEveShipSkill(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveShipSkill(ctx, arg.ShipTypeID, arg.Rank)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveType(args ...storage.CreateEveTypeParams) *app.EveType {
	var arg storage.CreateEveTypeParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_types", "id", startIDInventoryType))
	}
	x := f.CreateEveGroup(storage.CreateEveGroupParams{ID: arg.GroupID})
	arg.GroupID = x.ID
	if arg.Capacity == 0 {
		arg.Capacity = rand.Float32() * 1_000_000
	}
	if arg.Mass == 0 {
		arg.Mass = rand.Float32() * 10_000_000_000
	}
	if arg.Name == "" {
		arg.Name = fake.ProductName()
	}
	if arg.Description == "" {
		arg.Description = fake.Paragraph()
	}
	if arg.PortionSize == 0 {
		arg.PortionSize = 1
	}
	if arg.Radius == 0 {
		arg.Radius = rand.Float32() * 10_000
	}
	if arg.Volume == 0 {
		arg.Volume = rand.Float32() * 10_000_000
	}
	o, err := f.st.GetOrCreateEveType(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveTypeDogmaAttribute(args ...storage.CreateEveTypeDogmaAttributeParams) *app.EveTypeDogmaAttribute {
	var arg storage.CreateEveTypeDogmaAttributeParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.EveTypeID == 0 {
		x := f.CreateEveType()
		arg.EveTypeID = x.ID
	}
	if arg.DogmaAttributeID == 0 {
		x := f.CreateEveDogmaAttribute()
		arg.DogmaAttributeID = x.ID
	}
	if arg.Value == 0 {
		arg.Value = rand.Float32() * 10_000
	}
	if err := f.st.CreateEveTypeDogmaAttribute(ctx, arg); err != nil {
		panic(err)
	}
	v, err := f.st.GetEveTypeDogmaAttribute(ctx, arg.EveTypeID, arg.DogmaAttributeID)
	if err != nil {
		panic(err)
	}
	et, err := f.st.GetEveType(ctx, arg.EveTypeID)
	if err != nil {
		panic(err)
	}
	da, err := f.st.GetEveDogmaAttribute(ctx, arg.DogmaAttributeID)
	if err != nil {
		panic(err)
	}
	o := &app.EveTypeDogmaAttribute{
		EveType:        et,
		DogmaAttribute: da,
		Value:          v,
	}
	return o
}

func (f Factory) CreateEveDogmaAttribute(args ...storage.CreateEveDogmaAttributeParams) *app.EveDogmaAttribute {
	var arg storage.CreateEveDogmaAttributeParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_dogma_attributes", "id", 1))
	}
	if arg.DefaultValue == 0 {
		arg.DefaultValue = rand.Float32() * 10_000
	}
	if arg.Description == "" {
		arg.Description = fmt.Sprintf("Description #%d", arg.ID)
	}
	if arg.DisplayName == "" {
		arg.DisplayName = fmt.Sprintf("Display Name #%d", arg.ID)
	}
	if arg.IconID == 0 {
		arg.IconID = rand.Int32N(100_000)
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("Name #%d", arg.ID)
	}
	if arg.UnitID == 0 {
		arg.UnitID = app.EveUnitID(rand.IntN(120))
	}
	o, err := f.st.CreateEveDogmaAttribute(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveRegion(args ...storage.CreateEveRegionParams) *app.EveRegion {
	var arg storage.CreateEveRegionParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_regions", "id", startIDRegion))
	}
	if arg.Name == "" {
		arg.Name = fake.Continent()
	}
	o, err := f.st.CreateEveRegion(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveConstellation(args ...storage.CreateEveConstellationParams) *app.EveConstellation {
	var arg storage.CreateEveConstellationParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_constellations", "id", startIDConstellation))
	}
	if arg.Name == "" {
		arg.Name = fake.Country()
	}
	if arg.RegionID == 0 {
		x := f.CreateEveRegion()
		arg.RegionID = x.ID
	}
	err := f.st.CreateEveConstellation(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveConstellation(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveSolarSystem(args ...storage.CreateEveSolarSystemParams) *app.EveSolarSystem {
	var arg storage.CreateEveSolarSystemParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_solar_systems", "id", startIDSolarSystem))
	}
	if arg.Name == "" {
		arg.Name = fake.City()
	}
	if arg.ConstellationID == 0 {
		x := f.CreateEveConstellation()
		arg.ConstellationID = x.ID
	}
	if arg.SecurityStatus == 0 {
		arg.SecurityStatus = rand.Float32()*10 - 5
	}
	err := f.st.CreateEveSolarSystem(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveSolarSystem(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEvePlanet(args ...storage.CreateEvePlanetParams) *app.EvePlanet {
	var arg storage.CreateEvePlanetParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_planets", "id", startIDCelestials))
	}
	if arg.Name == "" {
		arg.Name = fake.Street()
	}
	if arg.SolarSystemID == 0 {
		x := f.CreateEveSolarSystem()
		arg.SolarSystemID = x.ID
	}
	if arg.TypeID == 0 {
		x := f.CreateEveType()
		arg.TypeID = x.ID
	}
	err := f.st.CreateEvePlanet(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEvePlanet(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateEveMoon(args ...storage.CreateEveMoonParams) *app.EveMoon {
	var arg storage.CreateEveMoonParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_moons", "id", startIDCelestials))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s %s", fake.Color(), fake.Street())
	}
	if arg.SolarSystemID == 0 {
		x := f.CreateEveSolarSystem()
		arg.SolarSystemID = x.ID
	}
	err := f.st.CreateEveMoon(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetEveMoon(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return o
}

// TODO: Refactor to storage.CreateEveRaceParams

func (f Factory) CreateEveRace(args ...app.EveRace) *app.EveRace {
	var arg app.EveRace
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_races", "id", startIDOther))
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s #%d", fake.JobTitle(), arg.ID)
	}
	if arg.Description == "" {
		arg.Description = fake.Paragraph()
	}
	arg2 := storage.CreateEveRaceParams{
		ID:          arg.ID,
		Description: arg.Description,
		Name:        arg.Name,
	}
	r, err := f.st.CreateEveRace(ctx, arg2)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateEveSchematic(args ...storage.CreateEveSchematicParams) *app.EveSchematic {
	var arg storage.CreateEveSchematicParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = int32(f.calcNewID("eve_schematics", "id", 1))
	}
	if arg.Name == "" {
		arg.Name = fake.ProductName()
	}
	r, err := f.st.CreateEveSchematic(ctx, arg)
	if err != nil {
		panic(err)
	}
	return r
}

func (f Factory) CreateEveLocationStructure(args ...storage.UpdateOrCreateLocationParams) *app.EveLocation {
	return f.createEveLocationStructure(startIDStructure, app.EveCategoryStructure, false, args...)
}

func (f Factory) CreateEveLocationStation(args ...storage.UpdateOrCreateLocationParams) *app.EveLocation {
	return f.createEveLocationStructure(startIDStation, app.EveCategoryStation, false, args...)
}

func (f Factory) CreateEveLocationEmptyStructure(args ...storage.UpdateOrCreateLocationParams) *app.EveLocation {
	return f.createEveLocationStructure(startIDStructure, app.EveCategoryStructure, true, args...)
}

func (f Factory) createEveLocationStructure(startID int64, categoryID int32, isEmpty bool, args ...storage.UpdateOrCreateLocationParams) *app.EveLocation {
	var arg storage.UpdateOrCreateLocationParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.ID == 0 {
		arg.ID = f.calcNewID("eve_locations", "id", startID)
	}
	if arg.Name == "" {
		arg.Name = fake.Color() + " " + fake.Brand()
	}
	if !isEmpty && arg.SolarSystemID.IsEmpty() {
		x := f.CreateEveSolarSystem()
		arg.SolarSystemID = optional.New(x.ID)
	}
	if !isEmpty && arg.OwnerID.IsEmpty() {
		x := f.CreateEveEntityCorporation()
		arg.OwnerID = optional.New(x.ID)
	}
	if !isEmpty && arg.TypeID.IsEmpty() {
		ec, err := f.st.GetEveCategory(ctx, categoryID)
		if err != nil {
			if errors.Is(err, app.ErrNotFound) {
				ec = f.CreateEveCategory(storage.CreateEveCategoryParams{ID: categoryID})
			} else {
				panic(err)
			}
		}
		eg := f.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: ec.ID})
		et := f.CreateEveType(storage.CreateEveTypeParams{GroupID: eg.ID})
		arg.TypeID = optional.New(et.ID)
	}
	if arg.UpdatedAt.IsZero() {
		arg.UpdatedAt = time.Now().UTC()
	}
	err := f.st.UpdateOrCreateEveLocation(ctx, arg)
	if err != nil {
		panic(err)
	}
	x, err := f.st.GetLocation(ctx, arg.ID)
	if err != nil {
		panic(err)
	}
	return x
}

func (f Factory) CreateEveMarketPrice(args ...storage.UpdateOrCreateEveMarketPriceParams) *app.EveMarketPrice {
	var arg storage.UpdateOrCreateEveMarketPriceParams
	ctx := context.Background()
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.TypeID == 0 {
		arg.TypeID = int32(f.calcNewID("eve_market_price", "type_id", 1))
	}
	if arg.AdjustedPrice == 0 {
		arg.AdjustedPrice = rand.Float64() * 100_000
	}
	if arg.AveragePrice == 0 {
		arg.AveragePrice = rand.Float64() * 100_000
	}
	o, err := f.st.UpdateOrCreateEveMarketPrice(ctx, arg)
	if err != nil {
		panic(err)
	}
	return o
}

func (f Factory) CreateStructureService(args ...storage.CreateStructureServiceParams) *app.StructureService {
	ctx := context.Background()
	var arg storage.CreateStructureServiceParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.CorporationStructureID == 0 {
		x := f.CreateCorporationStructure()
		arg.CorporationStructureID = x.ID
	}
	if arg.State == app.StructureServiceStateUndefined {
		arg.State = app.StructureServiceStateOnline
	}
	if arg.Name == "" {
		arg.Name = fmt.Sprintf("%s #%d", fake.Color(), generateSequenceID())
	}
	err := f.st.CreateStructureService(ctx, arg)
	if err != nil {
		panic(err)
	}
	o, err := f.st.GetStructureService(ctx, arg.CorporationStructureID, arg.Name)
	if err != nil {
		panic(err)
	}
	return o
}

func (f *Factory) CreateToken(args ...app.Token) *app.Token {
	o := &app.Token{
		AccessToken:   "AccessToken",
		CharacterID:   42,
		CharacterName: "Bruce Wayne",
		ExpiresAt:     time.Now().Add(20 * time.Minute).UTC(),
		RefreshToken:  "RefreshToken",
		Scopes:        []string{},
		TokenType:     "Character",
	}
	if len(args) == 0 {
		return o
	}
	a := args[0]
	if a.AccessToken != "" {
		o.AccessToken = a.AccessToken
	}
	if a.RefreshToken != "" {
		o.RefreshToken = a.RefreshToken
	}
	if a.CharacterName != "" {
		o.CharacterName = a.CharacterName
	}
	if a.CharacterID != 0 {
		o.CharacterID = a.CharacterID
	}
	if !a.ExpiresAt.IsZero() {
		o.ExpiresAt = a.ExpiresAt
	}
	if len(a.Scopes) > 0 {
		o.Scopes = a.Scopes
	}
	return o
}

func (f *Factory) calcNewID(table, idField string, start int64) int64 {
	if start < 1 {
		panic("start must be a positive number")
	}
	var vMax sql.NullInt64
	if err := f.dbRO.QueryRow(fmt.Sprintf("SELECT MAX(%s) FROM %s;", idField, table)).Scan(&vMax); err != nil {
		panic(err)
	}
	return max(vMax.Int64+1, start)
}

func (f *Factory) calcNewIDWithCharacter(table, idField string, characterID int32) int64 {
	var max sql.NullInt64
	sql := fmt.Sprintf("SELECT MAX(%s) FROM %s WHERE character_id = ?;", idField, table)
	if err := f.dbRO.QueryRow(sql, characterID).Scan(&max); err != nil {
		panic(err)
	}
	return max.Int64 + 1
}

func (f *Factory) calcNewIDWithCorporation(table, idField string, corporationID int32) int64 {
	var max sql.NullInt64
	sql := fmt.Sprintf("SELECT MAX(%s) FROM %s WHERE corporation_id = ?;", idField, table)
	if err := f.dbRO.QueryRow(sql, corporationID).Scan(&max); err != nil {
		panic(err)
	}
	return max.Int64 + 1
}

func (f *Factory) calcNewIDWithParam(table, idField, whereField string, whereValue int64) int64 {
	var max sql.NullInt64
	sql := fmt.Sprintf("SELECT MAX(%s) FROM %s WHERE %s = ?;", idField, table, whereField)
	if err := f.dbRO.QueryRow(sql, whereValue).Scan(&max); err != nil {
		panic(err)
	}
	return max.Int64 + 1
}

func (f *Factory) calcNewIDWithParams(table, idField string, clauses map[string]any) int64 {
	var max sql.NullInt64
	parts := make([]string, 0)
	for f, v := range clauses {
		parts = append(parts, fmt.Sprintf("%s = %v", f, v))
	}
	clausesStr := strings.Join(parts, " AND ")
	sql := fmt.Sprintf("SELECT MAX(%s) FROM %s WHERE %s;", idField, table, clausesStr)
	if err := f.dbRO.QueryRow(sql).Scan(&max); err != nil {
		panic(err)
	}
	return max.Int64 + 1
}

func calcContentHash(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b2 := md5.Sum(b)
	hash := hex.EncodeToString(b2[:])
	return hash, nil
}

// func generateUniqueHash() string {
// 	currentTime := time.Now().UnixNano()
// 	s := fmt.Sprintf("%d-%d", currentTime, rand.IntN(1_000_000_000_000))
// 	b2 := md5.Sum([]byte(s))
// 	hash := hex.EncodeToString(b2[:])
// 	return hash
// }

var uniqueID atomic.Int64

func generateSequenceID() int {
	return int(uniqueID.Add(1))
}
