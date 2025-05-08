// Package evenotification contains the business logic for dealing with Eve Online notifications.
// It defines the notification types and related categories
// and provides a service for rendering notifications titles and bodies.
package evenotification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type optionalString = optional.Optional[string]
type setInt32 = set.Set[int32]

// notificationRenderer represents the interface every notification renderer needs to confirm with.
type notificationRenderer interface {
	// entityIDs returns the Entity IDs used by a notification (if any).
	entityIDs(text string) (setInt32, error)
	// render returns the rendered title and body for a notification.
	render(ctx context.Context, text string, timestamp time.Time) (optionalString, optionalString, error)
	// setEveUniverse initialized access to the EveUniverseService service and must be called before render().
	setEveUniverse(*eveuniverseservice.EveUniverseService)
}

// baseRenderer represents the base renderer for all notification types.
//
// Each notification type has a renderer which can produce the title and string for a notification.
// In addition the renderer can return the Entity IDs of a notification,
// which allows refetching Entities for multile notifications in bulk before rendering.
//
// All rendered should embed baseRenderer and implement the render method.
// Renderers that want to return Entity IDs must also overwrite entityIDs.
type baseRenderer struct {
	eus *eveuniverseservice.EveUniverseService
}

func (br *baseRenderer) setEveUniverse(eus *eveuniverseservice.EveUniverseService) {
	br.eus = eus
}

// entityIDs returns the Entity IDs used by a notification (if any).
//
// Must be overwritten by a notification rendered that want to return IDs.
func (br baseRenderer) entityIDs(_ string) (setInt32, error) {
	return setInt32{}, nil
}

// EveNotificationService is a service for rendering notifications.
type EveNotificationService struct {
	eus *eveuniverseservice.EveUniverseService
}

func New(eus *eveuniverseservice.EveUniverseService) *EveNotificationService {
	s := &EveNotificationService{eus: eus}
	return s
}

// EntityIDs returns the Entity IDs used in a notification.
// This is useful to resolve Entity IDs in bulk for all notifications,
// before rendering them one by one.
// Returns an empty set when notification does not use Entity IDs.
func (s *EveNotificationService) EntityIDs(type_, text string) (setInt32, error) {
	t := Type(type_)
	if t == "" {
		return setInt32{}, nil
	}
	r, ok := s.makeRenderer(t)
	if !ok {
		return setInt32{}, nil
	}
	return r.entityIDs(text)
}

// RenderESI renders title and body for all supported notification types and returns them.
// Returns empty title and body for unsupported notification types.
func (s *EveNotificationService) RenderESI(ctx context.Context, type_, text string, timestamp time.Time) (optionalString, optionalString, error) {
	t := Type(type_)
	if t == "" {
		return optionalString{}, optionalString{}, nil
	}
	r, ok := s.makeRenderer(t)
	if !ok {
		return optionalString{}, optionalString{}, nil
	}
	return r.render(ctx, text, time.Time{})
}

func (s *EveNotificationService) makeRenderer(type_ Type) (notificationRenderer, bool) {
	m := map[Type]notificationRenderer{
		// billing
		BillOutOfMoneyMsg:                  &billOutOfMoneyMsg{},
		BillPaidCorpAllMsg:                 &billPaidCorpAllMsg{},
		CorpAllBillMsg:                     &corpAllBillMsg{},
		InfrastructureHubBillAboutToExpire: &infrastructureHubBillAboutToExpire{},
		IHubDestroyedByBillFailure:         &iHubDestroyedByBillFailure{},
		// corporate
		CharAppAcceptMsg:       &charAppAcceptMsg{},
		CharAppRejectMsg:       &charAppRejectMsg{},
		CharAppWithdrawMsg:     &charAppWithdrawMsg{},
		CharLeftCorpMsg:        &charLeftCorpMsg{},
		CorpAppInvitedMsg:      &corpAppInvitedMsg{},
		CorpAppNewMsg:          &corpAppNewMsg{},
		CorpAppRejectCustomMsg: &corpAppRejectCustomMsg{},
		// moonmining
		MoonminingAutomaticFracture:   &moonminingAutomaticFracture{},
		MoonminingExtractionCancelled: &moonminingExtractionCancelled{},
		MoonminingExtractionFinished:  &moonminingExtractionFinished{},
		MoonminingExtractionStarted:   &moonminingExtractionStarted{},
		MoonminingLaserFired:          &moonminingLaserFired{},
		// orbital
		OrbitalAttacked:   &orbitalAttacked{},
		OrbitalReinforced: &orbitalReinforced{},
		// structures
		OwnershipTransferred:                      &ownershipTransferred{},
		StructureAnchoring:                        &structureAnchoring{},
		StructureDestroyed:                        &structureDestroyed{},
		StructureFuelAlert:                        &structureFuelAlert{},
		StructureImpendingAbandonmentAssetsAtRisk: &structureImpendingAbandonmentAssetsAtRisk{},
		StructureItemsDelivered:                   &structureItemsDelivered{},
		StructureItemsMovedToSafety:               &structureItemsMovedToSafety{},
		StructureLostArmor:                        &structureLostArmor{},
		StructureLostShields:                      &structureLostShields{},
		StructureOnline:                           &structureOnline{},
		StructureServicesOffline:                  &structureServicesOffline{},
		StructuresReinforcementChanged:            &structuresReinforcementChanged{},
		StructureUnanchoring:                      &structureUnanchoring{},
		StructureUnderAttack:                      &structureUnderAttack{},
		StructureWentHighPower:                    &structureWentHighPower{},
		StructureWentLowPower:                     &structureWentLowPower{},
		// sov
		EntosisCaptureStarted:      &entosisCaptureStarted{},
		SovAllClaimAcquiredMsg:     &sovAllClaimAcquiredMsg{},
		SovAllClaimLostMsg:         &sovAllClaimLostMsg{},
		SovCommandNodeEventStarted: &sovCommandNodeEventStarted{},
		SovStructureDestroyed:      &sovStructureDestroyed{},
		SovStructureReinforced:     &sovStructureReinforced{},
		// tower
		TowerAlertMsg:         &towerAlertMsg{},
		TowerResourceAlertMsg: &towerResourceAlertMsg{},
		// war
		AllWarSurrenderMsg:    &allWarSurrenderMsg{},
		CorpWarSurrenderMsg:   &corpWarSurrenderMsg{},
		DeclareWar:            &declareWar{},
		WarAdopted:            &warAdopted{},
		WarDeclared:           &warDeclared{},
		WarHQRemovedFromSpace: &warHQRemovedFromSpace{},
		WarInherited:          &warInherited{},
		WarInvalid:            &warInvalid{},
		WarRetractedByConcord: &warRetractedByConcord{},
	}
	r, found := m[type_]
	if !found {
		return nil, found
	}
	r.setEveUniverse(s.eus)
	return r, true
}

// fromLDAPTime converts an ldap time to golang time
func fromLDAPTime(ldap_dt int64) time.Time {
	return time.Unix((ldap_dt/10000000)-11644473600, 0).UTC()
}

// fromLDAPDuration converts an ldap duration to golang duration
func fromLDAPDuration(ldap_td int64) time.Duration {
	return time.Duration(ldap_td/10) * time.Microsecond
}

type dotlanType = uint

const (
	dotlanAlliance dotlanType = iota
	dotlanCorporation
	dotlanSolarSystem
	dotlanRegion
)

func makeDotLanProfileURL(name string, typ dotlanType) string {
	const baseURL = "https://evemaps.dotlan.net"
	var path string
	m := map[dotlanType]string{
		dotlanAlliance:    "alliance",
		dotlanCorporation: "corp",
		dotlanSolarSystem: "system",
		dotlanRegion:      "region",
	}
	path, ok := m[typ]
	if !ok {
		return name
	}
	name2 := strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s/%s/%s", baseURL, path, name2)
}

func makeSolarSystemLink(ess *app.EveSolarSystem) string {
	x := fmt.Sprintf(
		"%s (%s)",
		makeMarkDownLink(ess.Name, makeDotLanProfileURL(ess.Name, dotlanSolarSystem)),
		ess.Constellation.Region.Name,
	)
	return x
}

func makeCorporationLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanCorporation))
}

func makeAllianceLink(name string) string {
	if name == "" {
		return ""
	}
	return makeMarkDownLink(name, makeDotLanProfileURL(name, dotlanAlliance))
}

func makeEveWhoCharacterURL(id int32) string {
	return fmt.Sprintf("https://evewho.com/character/%d", id)
}

func makeEveEntityProfileLink(e *app.EveEntity) string {
	if e == nil {
		return ""
	}
	var url string
	switch e.Category {
	case app.EveEntityAlliance:
		url = makeDotLanProfileURL(e.Name, dotlanAlliance)
	case app.EveEntityCharacter:
		url = makeEveWhoCharacterURL(e.ID)
	case app.EveEntityCorporation:
		url = makeDotLanProfileURL(e.Name, dotlanCorporation)
	default:
		return e.Name
	}
	return makeMarkDownLink(e.Name, url)
}

func makeMarkDownLink(label, url string) string {
	return fmt.Sprintf("[%s](%s)", label, url)
}
