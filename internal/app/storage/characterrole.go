package storage

import (
	"context"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

var role2String = map[app.Role]string{
	app.RoleHangarTake3:             "hangar_take_3",
	app.RoleHangarTake7:             "hangar_take_7",
	app.RoleRentResearchFacility:    "rent_research_facility",
	app.RoleSkillPlanManager:        "skill_plan_manager",
	app.RoleTrader:                  "trader",
	app.RoleConfigEquipment:         "config_equipment",
	app.RoleContainerTake3:          "container_take_3",
	app.RoleContainerTake7:          "container_take_7",
	app.RoleHangarQuery1:            "hangar_query_1",
	app.RoleStationManager:          "station_manager",
	app.RoleAccountTake1:            "account_take_1",
	app.RoleAccountTake4:            "account_take_4",
	app.RoleAccountant:              "accountant",
	app.RoleContainerTake4:          "container_take_4",
	app.RoleContainerTake6:          "container_take_6",
	app.RoleDeliveriesTake:          "deliveries_take",
	app.RoleHangarTake1:             "hangar_take_1",
	app.RoleRentFactoryFacility:     "rent_factory_facility",
	app.RoleFittingManager:          "fitting_manager",
	app.RoleAuditor:                 "auditor",
	app.RoleConfigStarbaseEquipment: "config_starbase_equipment",
	app.RoleHangarQuery3:            "hangar_query_3",
	app.RoleHangarQuery7:            "hangar_query_7",
	app.RoleHangarTake6:             "hangar_take_6",
	app.RoleRentOffice:              "rent_office",
	app.RoleStarbaseDefenseOperator: "starbase_defense_operator",
	app.RoleAccountTake2:            "account_take_2",
	app.RoleAccountTake3:            "account_take_3",
	app.RoleAccountTake6:            "account_take_6",
	app.RoleCommunicationsOfficer:   "communications_officer",
	app.RoleDeliveriesQuery:         "deliveries_query",
	app.RoleHangarQuery5:            "hangar_query_5",
	app.RoleHangarTake4:             "hangar_take_4",
	app.RolePersonnelManager:        "personnel_manager",
	app.RoleDeliveriesContainerTake: "deliveries_container_take",
	app.RoleDirector:                "director",
	app.RoleJuniorAccountant:        "junior_accountant",
	app.RoleProjectManager:          "project_manager",
	app.RoleAccountTake5:            "account_take_5",
	app.RoleHangarTake2:             "hangar_take_2",
	app.RoleHangarTake5:             "hangar_take_5",
	app.RoleSecurityOfficer:         "security_officer",
	app.RoleStarbaseFuelTechnician:  "starbase_fuel_technician",
	app.RoleBrandManager:            "brand_manager",
	app.RoleContainerTake1:          "container_take_1",
	app.RoleContainerTake2:          "container_take_2",
	app.RoleContainerTake5:          "container_take_5",
	app.RoleDiplomat:                "diplomat",
	app.RoleFactoryManager:          "factory_manager",
	app.RoleHangarQuery2:            "hangar_query_2",
	app.RoleHangarQuery4:            "hangar_query_4",
	app.RoleAccountTake7:            "account_take_7",
	app.RoleContractManager:         "contract_manager",
	app.RoleHangarQuery6:            "hangar_query_6",
}

var string2Role = map[string]app.Role{}

func init() {
	for k, v := range role2String {
		string2Role[v] = k
	}
}

func (st *Storage) ListCharacterRoles(ctx context.Context, characterID int32) (set.Set[app.Role], error) {
	s, err := st.qRO.ListCharacterRoles(ctx, int64(characterID))
	if err != nil {
		return set.Set[app.Role]{}, fmt.Errorf("list roles for character %d: %w", characterID, err)
	}
	r := set.Collect(xiter.Map(slices.Values(s), func(x string) app.Role {
		return string2Role[x]
	}))
	return r, err
}

func (st *Storage) UpdateCharacterRoles(ctx context.Context, characterID int32, roles set.Set[app.Role]) error {
	incoming := roles2names(roles)
	s, err := st.qRO.ListCharacterRoles(ctx, int64(characterID))
	if err != nil {
		return err
	}
	current := set.Of(s...)
	added := set.Difference(incoming, current)
	for s := range added.All() {
		arg := queries.CreateCharacterRoleParams{
			CharacterID: int64(characterID),
			Name:        s,
		}
		if err := st.qRW.CreateCharacterRole(ctx, arg); err != nil {
			return fmt.Errorf("create character role: %+v: %w", arg, err)
		}
	}
	removed := set.Difference(current, incoming)
	for s := range removed.All() {
		arg := queries.DeleteCharacterRoleParams{
			CharacterID: int64(characterID),
			Name:        s,
		}
		if err := st.qRW.DeleteCharacterRole(ctx, arg); err != nil {
			return fmt.Errorf("delete character role: %+v: %w", arg, err)
		}
	}
	return nil
}

func roles2names(roles set.Set[app.Role]) set.Set[string] {
	return set.Collect(xiter.Map(roles.All(), func(r app.Role) string {
		return role2String[r]
	}))
}
