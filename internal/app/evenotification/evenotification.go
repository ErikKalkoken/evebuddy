// Package evenotification contains the business logic for dealing with Eve Online notifications.
//
// It defines the notification types and related categories
// and provides a service for rendering notifications titles and bodies.
package evenotification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type EveUniverseService interface {
	GetOrCreateEntityESI(ctx context.Context, id int32) (*app.EveEntity, error)
	GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error)
	GetOrCreateMoonESI(ctx context.Context, id int32) (*app.EveMoon, error)
	GetOrCreatePlanetESI(ctx context.Context, id int32) (*app.EvePlanet, error)
	GetOrCreateSolarSystemESI(ctx context.Context, id int32) (*app.EveSolarSystem, error)
	GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error)
	ToEntities(ctx context.Context, ids set.Set[int32]) (map[int32]*app.EveEntity, error)
}

type setInt32 = set.Set[int32]

// notificationRenderer represents the interface every notification renderer needs to confirm with.
type notificationRenderer interface {
	// entityIDs returns the Entity IDs used by a notification (if any).
	entityIDs(text string) (setInt32, error)
	// render returns the rendered title and body for a notification.
	render(ctx context.Context, text string, timestamp time.Time) (string, string, error)
	// setEveUniverse initialized access to the EveUniverseService service and must be called before render().
	setEveUniverse(EveUniverseService)
}

// baseRenderer represents the base renderer for all notification types.
//
// Each notification type has a renderer which can produce the title and string for a notification.
// In addition the renderer can return the Entity IDs of a notification,
// which allows refetching Entities for multiple notifications in bulk before rendering.
//
// All rendered should embed baseRenderer and implement the render method.
// Renderers that want to return Entity IDs must also overwrite entityIDs.
type baseRenderer struct {
	eus EveUniverseService
}

func (br *baseRenderer) setEveUniverse(eus EveUniverseService) {
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
	eus EveUniverseService
}

func New(eus EveUniverseService) *EveNotificationService {
	s := &EveNotificationService{eus: eus}
	return s
}

// EntityIDs returns the Entity IDs used in a notification.
// This is useful to resolve Entity IDs in bulk for all notifications,
// before rendering them one by one.
// Returns an empty set when notification does not use Entity IDs.
// Returns [app.ErrNotFound] for unsupported notification types.
func (s *EveNotificationService) EntityIDs(nt app.EveNotificationType, text string) (setInt32, error) {
	r, found := s.makeRenderer(app.EveNotificationType(nt))
	if !found {
		return setInt32{}, app.ErrNotFound
	}
	return r.entityIDs(text)
}

// RenderESI renders title and body for all supported notification types and returns them.
// Returns [app.ErrNotFound] for unsupported notification types.
func (s *EveNotificationService) RenderESI(ctx context.Context, nt app.EveNotificationType, text string, timestamp time.Time) (title string, body string, err error) {
	r, found := s.makeRenderer(app.EveNotificationType(nt))
	if !found {
		return "", "", app.ErrNotFound
	}
	title, body, err = r.render(ctx, text, timestamp)
	if err != nil {
		return "", "", err
	}
	return title, body, nil
}

func (s *EveNotificationService) makeRenderer(type_ app.EveNotificationType) (notificationRenderer, bool) {
	var r notificationRenderer
	switch type_ {
	// billing
	case app.BillOutOfMoneyMsg:
		r = new(billOutOfMoneyMsg)
	case app.BillPaidCorpAllMsg:
		r = new(billPaidCorpAllMsg)
	case app.CorpAllBillMsg:
		r = new(corpAllBillMsg)
	case app.InfrastructureHubBillAboutToExpire:
		r = new(infrastructureHubBillAboutToExpire)
	case app.IHubDestroyedByBillFailure:
		r = new(iHubDestroyedByBillFailure)
	// corporate
	case app.CharAppAcceptMsg:
		r = new(charAppAcceptMsg)
	case app.CharAppRejectMsg:
		r = new(charAppRejectMsg)
	case app.CharAppWithdrawMsg:
		r = new(charAppWithdrawMsg)
	case app.CharLeftCorpMsg:
		r = new(charLeftCorpMsg)
	case app.CorpAppInvitedMsg:
		r = new(corpAppInvitedMsg)
	case app.CorpAppNewMsg:
		r = new(corpAppNewMsg)
	case app.CorpAppRejectCustomMsg:
		r = new(corpAppRejectCustomMsg)
	// moonmining
	case app.MoonminingAutomaticFracture:
		r = new(moonminingAutomaticFracture)
	case app.MoonminingExtractionCancelled:
		r = new(moonminingExtractionCancelled)
	case app.MoonminingExtractionFinished:
		r = new(moonminingExtractionFinished)
	case app.MoonminingExtractionStarted:
		r = new(moonminingExtractionStarted)
	case app.MoonminingLaserFired:
		r = new(moonminingLaserFired)
	// orbital
	case app.OrbitalAttacked:
		r = new(orbitalAttacked)
	case app.OrbitalReinforced:
		r = new(orbitalReinforced)
	// structures
	case app.MercenaryDenAttacked:
		r = new(mercenaryDenAttacked)
	case app.MercenaryDenReinforced:
		r = new(mercenaryDenReinforced)
	case app.OwnershipTransferred:
		r = new(ownershipTransferred)
	case app.StructureAnchoring:
		r = new(structureAnchoring)
	case app.StructureDestroyed:
		r = new(structureDestroyed)
	case app.StructureFuelAlert:
		r = new(structureFuelAlert)
	case app.StructureImpendingAbandonmentAssetsAtRisk:
		r = new(structureImpendingAbandonmentAssetsAtRisk)
	case app.StructureItemsDelivered:
		r = new(structureItemsDelivered)
	case app.StructureItemsMovedToSafety:
		r = new(structureItemsMovedToSafety)
	case app.StructureLostArmor:
		r = new(structureLostArmor)
	case app.StructureLostShields:
		r = new(structureLostShields)
	case app.StructureOnline:
		r = new(structureOnline)
	case app.StructureServicesOffline:
		r = new(structureServicesOffline)
	case app.StructuresReinforcementChanged:
		r = new(structuresReinforcementChanged)
	case app.StructureUnanchoring:
		r = new(structureUnanchoring)
	case app.StructureUnderAttack:
		r = new(structureUnderAttack)
	case app.StructureWentHighPower:
		r = new(structureWentHighPower)
	case app.StructureWentLowPower:
		r = new(structureWentLowPower)
	// sov
	case app.EntosisCaptureStarted:
		r = new(entosisCaptureStarted)
	case app.SovAllClaimAcquiredMsg:
		r = new(sovAllClaimAcquiredMsg)
	case app.SovAllClaimLostMsg:
		r = new(sovAllClaimLostMsg)
	case app.SovCommandNodeEventStarted:
		r = new(sovCommandNodeEventStarted)
	case app.SovStructureDestroyed:
		r = new(sovStructureDestroyed)
	case app.SovStructureReinforced:
		r = new(sovStructureReinforced)
	// tower
	case app.TowerAlertMsg:
		r = new(towerAlertMsg)
	case app.TowerResourceAlertMsg:
		r = new(towerResourceAlertMsg)
	// war
	case app.AllWarSurrenderMsg:
		r = new(allWarSurrenderMsg)
	case app.CorpWarSurrenderMsg:
		r = new(corpWarSurrenderMsg)
	case app.DeclareWar:
		r = new(declareWar)
	case app.WarAdopted:
		r = new(warAdopted)
	case app.WarDeclared:
		r = new(warDeclared)
	case app.WarHQRemovedFromSpace:
		r = new(warHQRemovedFromSpace)
	case app.WarInherited:
		r = new(warInherited)
	case app.WarInvalid:
		r = new(warInvalid)
	case app.WarRetractedByConcord:
		r = new(warRetractedByConcord)
	default:
		return nil, false
	}
	r.setEveUniverse(s.eus)
	return r, true
}

// fromLDAPTime converts an ldap time to golang time
func fromLDAPTime(ldapTime int64) time.Time {
	return time.Unix((ldapTime/10000000)-11644473600, 0).UTC()
}

// fromLDAPDuration converts an ldap duration to golang duration
func fromLDAPDuration(ldapDuration int64) time.Duration {
	return time.Duration(ldapDuration/10) * time.Microsecond
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

func makeEveEntityProfileLink(o *app.EveEntity) string {
	if o == nil {
		return "?"
	}
	var url string
	switch o.Category {
	case app.EveEntityAlliance:
		url = makeDotLanProfileURL(o.Name, dotlanAlliance)
	case app.EveEntityCharacter:
		url = makeEveWhoCharacterURL(o.ID)
	case app.EveEntityCorporation:
		url = makeDotLanProfileURL(o.Name, dotlanCorporation)
	default:
		return o.Name
	}
	return makeMarkDownLink(o.Name, url)
}

func makeMarkDownLink(label, url string) string {
	return fmt.Sprintf("[%s](%s)", label, url)
}
