package characterservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
	"github.com/antihax/goesi/esi"
)

func (s *CharacterService) ListRoles(ctx context.Context, characterID int32) ([]app.CharacterRole, error) {
	granted, err := s.st.ListCharacterRoles(ctx, characterID)
	if err != nil {
		return nil, err
	}
	rolesSorted := slices.SortedFunc(app.RolesAll(), func(a, b app.Role) int {
		return strings.Compare(a.String(), b.String())
	})
	roles := make([]app.CharacterRole, 0)
	if granted.Contains(app.RoleDirector) {
		roles = append(roles, app.CharacterRole{
			CharacterID: characterID,
			Role:        app.RoleDirector,
			Granted:     true,
		})
		return roles, nil
	}
	for _, r := range rolesSorted {
		roles = append(roles, app.CharacterRole{
			CharacterID: characterID,
			Role:        r,
			Granted:     granted.Contains(r),
		})
	}
	return roles, nil
}

// Roles
func (s *CharacterService) updateRolesESI(ctx context.Context, arg app.CharacterSectionUpdateParams) (bool, error) {
	if arg.Section != app.SectionCharacterRoles {
		return false, fmt.Errorf("wrong section for update %s: %w", arg.Section, app.ErrInvalid)
	}
	roleMap := map[string]app.Role{
		"Account_Take_1":            app.RoleAccountTake1,
		"Account_Take_2":            app.RoleAccountTake2,
		"Account_Take_3":            app.RoleAccountTake3,
		"Account_Take_4":            app.RoleAccountTake4,
		"Account_Take_5":            app.RoleAccountTake5,
		"Account_Take_6":            app.RoleAccountTake6,
		"Account_Take_7":            app.RoleAccountTake7,
		"Accountant":                app.RoleAccountant,
		"Auditor":                   app.RoleAuditor,
		"Brand_Manager":             app.RoleBrandManager,
		"Communications_Officer":    app.RoleCommunicationsOfficer,
		"Config_Equipment":          app.RoleConfigEquipment,
		"Config_Starbase_Equipment": app.RoleConfigStarbaseEquipment,
		"Container_Take_1":          app.RoleContainerTake1,
		"Container_Take_2":          app.RoleContainerTake2,
		"Container_Take_3":          app.RoleContainerTake3,
		"Container_Take_4":          app.RoleContainerTake4,
		"Container_Take_5":          app.RoleContainerTake5,
		"Container_Take_6":          app.RoleContainerTake6,
		"Container_Take_7":          app.RoleContainerTake7,
		"Contract_Manager":          app.RoleContractManager,
		"Deliveries_Container_Take": app.RoleDeliveriesContainerTake,
		"Deliveries_Query":          app.RoleDeliveriesQuery,
		"Deliveries_Take":           app.RoleDeliveriesTake,
		"Diplomat":                  app.RoleDiplomat,
		"Director":                  app.RoleDirector,
		"Factory_Manager":           app.RoleFactoryManager,
		"Fitting_Manager":           app.RoleFittingManager,
		"Hangar_Query_1":            app.RoleHangarQuery1,
		"Hangar_Query_2":            app.RoleHangarQuery2,
		"Hangar_Query_3":            app.RoleHangarQuery3,
		"Hangar_Query_4":            app.RoleHangarQuery4,
		"Hangar_Query_5":            app.RoleHangarQuery5,
		"Hangar_Query_6":            app.RoleHangarQuery6,
		"Hangar_Query_7":            app.RoleHangarQuery7,
		"Hangar_Take_1":             app.RoleHangarTake1,
		"Hangar_Take_2":             app.RoleHangarTake2,
		"Hangar_Take_3":             app.RoleHangarTake3,
		"Hangar_Take_4":             app.RoleHangarTake4,
		"Hangar_Take_5":             app.RoleHangarTake5,
		"Hangar_Take_6":             app.RoleHangarTake6,
		"Hangar_Take_7":             app.RoleHangarTake7,
		"Junior_Accountant":         app.RoleJuniorAccountant,
		"Personnel_Manager":         app.RolePersonnelManager,
		"Project_Manager":           app.RoleProjectManager,
		"Rent_Factory_Facility":     app.RoleRentFactoryFacility,
		"Rent_Office":               app.RoleRentOffice,
		"Rent_Research_Facility":    app.RoleRentResearchFacility,
		"Security_Officer":          app.RoleSecurityOfficer,
		"Skill_Plan_Manager":        app.RoleSkillPlanManager,
		"Starbase_Defense_Operator": app.RoleStarbaseDefenseOperator,
		"Starbase_Fuel_Technician":  app.RoleStarbaseFuelTechnician,
		"Station_Manager":           app.RoleStationManager,
		"Trader":                    app.RoleTrader,
	}

	return s.updateSectionIfChanged(
		ctx, arg,
		func(ctx context.Context, characterID int32) (any, error) {
			roles, _, err := xesi.RateLimited(ctx, "GetCharactersCharacterIdRoles", func() (esi.GetCharactersCharacterIdRolesOk, *http.Response, error) {
				return s.esiClient.ESI.CharacterApi.GetCharactersCharacterIdRoles(ctx, characterID, nil)
			})
			if err != nil {
				return false, err
			}
			return roles, nil
		},
		func(ctx context.Context, characterID int32, data any) error {
			r := data.(esi.GetCharactersCharacterIdRolesOk)
			var roles set.Set[app.Role]
			for _, n := range r.Roles {
				r, ok := roleMap[n]
				if !ok {
					slog.Warn("received unknown role from ESI", "characterID", characterID, "role", n)
				}
				roles.Add(r)
			}
			return s.st.UpdateCharacterRoles(ctx, characterID, roles)
		})
}
