# ESI rate limits

This document shows the current rate limits for the ESI API
and was generated from openAPI 3.1 spec with the compatibility date: 2025-11-06.

## Contents

- [Rate limit groups](#rate-limit-groups)
- [Operations by tag](#operations-by-tag)
- [Operations flat](#operations-flat)

## Rate limit groups

This section shows the current configuration of all active rate limit groups.

Group | Max tokens | Windows size | Average rate
-- | -- | -- | --
alliance-social | 300 | 900 | 0.33
char-contract | 600 | 900 | 0.67
char-detail | 600 | 900 | 0.67
char-industry | 600 | 900 | 0.67
char-killmail | 30 | 900 | 0.03
char-location | 1200 | 900 | 1.33
char-notification | 15 | 900 | 0.02
char-social | 600 | 900 | 0.67
char-wallet | 150 | 900 | 0.17
corp-contract | 600 | 900 | 0.67
corp-detail | 300 | 900 | 0.33
corp-industry | 600 | 900 | 0.67
corp-killmail | 30 | 900 | 0.03
corp-member | 300 | 900 | 0.33
corp-social | 300 | 900 | 0.33
corp-wallet | 300 | 900 | 0.33
factional-warfare | 150 | 900 | 0.17
fitting | 150 | 900 | 0.17
fleet | 1800 | 900 | 2
incursion | 150 | 900 | 0.17
industry | 150 | 900 | 0.17
insurance | 150 | 900 | 0.17
killmail | 3600 | 900 | 4
routes | 3600 | 900 | 4
sovereignty | 600 | 900 | 0.67
status | 600 | 900 | 0.67
ui | 900 | 900 | 1

The windows size is given in seconds. The average rate is given in requests per second.

## Operations by tag

This section shows the active rate limit group for all ESI operations.
An empty group entry means that the rate limit group for that operation has not yet been activated.
The operations are structured by tags.

### Alliance

Operation ID | Rate limit Group
-- | --
GetAlliances | -
GetAlliancesAllianceId | -
GetAlliancesAllianceIdCorporations | -
GetAlliancesAllianceIdIcons | -

### Assets

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdAssets | -
GetCorporationsCorporationIdAssets | -
PostCharactersCharacterIdAssetsLocations | -
PostCharactersCharacterIdAssetsNames | -
PostCorporationsCorporationIdAssetsLocations | -
PostCorporationsCorporationIdAssetsNames | -

### Calendar

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdCalendar | char-social
GetCharactersCharacterIdCalendarEventId | char-social
GetCharactersCharacterIdCalendarEventIdAttendees | char-social
PutCharactersCharacterIdCalendarEventId | char-social

### Character

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterId | -
GetCharactersCharacterIdAgentsResearch | char-industry
GetCharactersCharacterIdBlueprints | char-industry
GetCharactersCharacterIdCorporationhistory | -
GetCharactersCharacterIdFatigue | char-location
GetCharactersCharacterIdMedals | char-detail
GetCharactersCharacterIdNotifications | char-notification
GetCharactersCharacterIdNotificationsContacts | char-social
GetCharactersCharacterIdPortrait | char-detail
GetCharactersCharacterIdRoles | char-detail
GetCharactersCharacterIdStandings | char-social
GetCharactersCharacterIdTitles | char-detail
PostCharactersAffiliation | -
PostCharactersCharacterIdCspa | char-detail

### Clones

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdClones | char-location
GetCharactersCharacterIdImplants | char-detail

### Contacts

Operation ID | Rate limit Group
-- | --
DeleteCharactersCharacterIdContacts | char-social
GetAlliancesAllianceIdContacts | alliance-social
GetAlliancesAllianceIdContactsLabels | alliance-social
GetCharactersCharacterIdContacts | char-social
GetCharactersCharacterIdContactsLabels | char-social
GetCorporationsCorporationIdContacts | corp-social
GetCorporationsCorporationIdContactsLabels | corp-social
PostCharactersCharacterIdContacts | char-social
PutCharactersCharacterIdContacts | char-social

### Contracts

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdContracts | char-contract
GetCharactersCharacterIdContractsContractIdBids | char-contract
GetCharactersCharacterIdContractsContractIdItems | char-contract
GetContractsPublicBidsContractId | -
GetContractsPublicItemsContractId | -
GetContractsPublicRegionId | -
GetCorporationsCorporationIdContracts | corp-contract
GetCorporationsCorporationIdContractsContractIdBids | corp-contract
GetCorporationsCorporationIdContractsContractIdItems | corp-contract

### Corporation

Operation ID | Rate limit Group
-- | --
GetCorporationsCorporationId | -
GetCorporationsCorporationIdAlliancehistory | -
GetCorporationsCorporationIdBlueprints | corp-industry
GetCorporationsCorporationIdContainersLogs | -
GetCorporationsCorporationIdDivisions | corp-wallet
GetCorporationsCorporationIdFacilities | -
GetCorporationsCorporationIdIcons | -
GetCorporationsCorporationIdMedals | corp-detail
GetCorporationsCorporationIdMedalsIssued | corp-detail
GetCorporationsCorporationIdMembers | corp-member
GetCorporationsCorporationIdMembersLimit | corp-member
GetCorporationsCorporationIdMembersTitles | corp-member
GetCorporationsCorporationIdMembertracking | corp-member
GetCorporationsCorporationIdRoles | corp-member
GetCorporationsCorporationIdRolesHistory | corp-member
GetCorporationsCorporationIdShareholders | corp-detail
GetCorporationsCorporationIdStandings | corp-member
GetCorporationsCorporationIdStarbases | -
GetCorporationsCorporationIdStarbasesStarbaseId | -
GetCorporationsCorporationIdStructures | -
GetCorporationsCorporationIdTitles | corp-detail
GetCorporationsNpccorps | -

### Corporation Projects

Operation ID | Rate limit Group
-- | --
GetCorporationsProjectsContribution | -
GetCorporationsProjectsContributors | -
GetCorporationsProjectsDetail | -
GetCorporationsProjectsListing | -

### Dogma

Operation ID | Rate limit Group
-- | --
GetDogmaAttributes | -
GetDogmaAttributesAttributeId | -
GetDogmaDynamicItemsTypeIdItemId | -
GetDogmaEffects | -
GetDogmaEffectsEffectId | -

### Faction Warfare

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdFwStats | factional-warfare
GetCorporationsCorporationIdFwStats | factional-warfare
GetFwLeaderboards | factional-warfare
GetFwLeaderboardsCharacters | factional-warfare
GetFwLeaderboardsCorporations | factional-warfare
GetFwStats | factional-warfare
GetFwSystems | factional-warfare
GetFwWars | factional-warfare

### Fittings

Operation ID | Rate limit Group
-- | --
DeleteCharactersCharacterIdFittingsFittingId | fitting
GetCharactersCharacterIdFittings | fitting
PostCharactersCharacterIdFittings | fitting

### Fleets

Operation ID | Rate limit Group
-- | --
DeleteFleetsFleetIdMembersMemberId | fleet
DeleteFleetsFleetIdSquadsSquadId | fleet
DeleteFleetsFleetIdWingsWingId | fleet
GetCharactersCharacterIdFleet | fleet
GetFleetsFleetId | fleet
GetFleetsFleetIdMembers | fleet
GetFleetsFleetIdWings | fleet
PostFleetsFleetIdMembers | fleet
PostFleetsFleetIdWings | fleet
PostFleetsFleetIdWingsWingIdSquads | fleet
PutFleetsFleetId | fleet
PutFleetsFleetIdMembersMemberId | fleet
PutFleetsFleetIdSquadsSquadId | fleet
PutFleetsFleetIdWingsWingId | fleet

### Incursions

Operation ID | Rate limit Group
-- | --
GetIncursions | incursion

### Industry

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdIndustryJobs | char-industry
GetCharactersCharacterIdMining | char-industry
GetCorporationCorporationIdMiningExtractions | corp-industry
GetCorporationCorporationIdMiningObservers | corp-industry
GetCorporationCorporationIdMiningObserversObserverId | corp-industry
GetCorporationsCorporationIdIndustryJobs | corp-industry
GetIndustryFacilities | industry
GetIndustrySystems | industry

### Insurance

Operation ID | Rate limit Group
-- | --
GetInsurancePrices | insurance

### Killmails

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdKillmailsRecent | char-killmail
GetCorporationsCorporationIdKillmailsRecent | corp-killmail
GetKillmailsKillmailIdKillmailHash | killmail

### Location

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdLocation | char-location
GetCharactersCharacterIdOnline | char-location
GetCharactersCharacterIdShip | char-location

### Loyalty

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdLoyaltyPoints | char-wallet
GetLoyaltyStoresCorporationIdOffers | -

### Mail

Operation ID | Rate limit Group
-- | --
DeleteCharactersCharacterIdMailLabelsLabelId | char-social
DeleteCharactersCharacterIdMailMailId | char-social
GetCharactersCharacterIdMail | char-social
GetCharactersCharacterIdMailLabels | char-social
GetCharactersCharacterIdMailLists | char-social
GetCharactersCharacterIdMailMailId | char-social
PostCharactersCharacterIdMail | char-social
PostCharactersCharacterIdMailLabels | char-social
PutCharactersCharacterIdMailMailId | char-social

### Market

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdOrders | -
GetCharactersCharacterIdOrdersHistory | -
GetCorporationsCorporationIdOrders | -
GetCorporationsCorporationIdOrdersHistory | -
GetMarketsGroups | -
GetMarketsGroupsMarketGroupId | -
GetMarketsPrices | -
GetMarketsRegionIdHistory | -
GetMarketsRegionIdOrders | -
GetMarketsRegionIdTypes | -
GetMarketsStructuresStructureId | -

### Meta

Operation ID | Rate limit Group
-- | --
GetMetaChangelog | -
GetMetaCompatibilityDates | -
GetMetaStatus | -

### Planetary Interaction

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdPlanets | char-industry
GetCharactersCharacterIdPlanetsPlanetId | char-industry
GetCorporationsCorporationIdCustomsOffices | corp-industry
GetUniverseSchematicsSchematicId | -

### Routes

Operation ID | Rate limit Group
-- | --
PostRoute | routes

### Search

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdSearch | -

### Skills

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdAttributes | char-detail
GetCharactersCharacterIdSkillqueue | char-detail
GetCharactersCharacterIdSkills | char-detail

### Sovereignty

Operation ID | Rate limit Group
-- | --
GetSovereigntyCampaigns | sovereignty
GetSovereigntyMap | sovereignty
GetSovereigntyStructures | sovereignty

### Status

Operation ID | Rate limit Group
-- | --
GetStatus | status

### Universe

Operation ID | Rate limit Group
-- | --
GetUniverseAncestries | -
GetUniverseAsteroidBeltsAsteroidBeltId | -
GetUniverseBloodlines | -
GetUniverseCategories | -
GetUniverseCategoriesCategoryId | -
GetUniverseConstellations | -
GetUniverseConstellationsConstellationId | -
GetUniverseFactions | -
GetUniverseGraphics | -
GetUniverseGraphicsGraphicId | -
GetUniverseGroups | -
GetUniverseGroupsGroupId | -
GetUniverseMoonsMoonId | -
GetUniversePlanetsPlanetId | -
GetUniverseRaces | -
GetUniverseRegions | -
GetUniverseRegionsRegionId | -
GetUniverseStargatesStargateId | -
GetUniverseStarsStarId | -
GetUniverseStationsStationId | -
GetUniverseStructures | -
GetUniverseStructuresStructureId | -
GetUniverseSystemJumps | -
GetUniverseSystemKills | -
GetUniverseSystems | -
GetUniverseSystemsSystemId | -
GetUniverseTypes | -
GetUniverseTypesTypeId | -
PostUniverseIds | -
PostUniverseNames | -

### User Interface

Operation ID | Rate limit Group
-- | --
PostUiAutopilotWaypoint | ui
PostUiOpenwindowContract | ui
PostUiOpenwindowInformation | ui
PostUiOpenwindowMarketdetails | ui
PostUiOpenwindowNewmail | ui

### Wallet

Operation ID | Rate limit Group
-- | --
GetCharactersCharacterIdWallet | char-wallet
GetCharactersCharacterIdWalletJournal | char-wallet
GetCharactersCharacterIdWalletTransactions | char-wallet
GetCorporationsCorporationIdWallets | corp-wallet
GetCorporationsCorporationIdWalletsDivisionJournal | corp-wallet
GetCorporationsCorporationIdWalletsDivisionTransactions | corp-wallet

### Wars

Operation ID | Rate limit Group
-- | --
GetWars | killmail
GetWarsWarId | killmail
GetWarsWarIdKillmails | killmail

## Operations flat

This section shows the active rate limit group for all ESI operations.
An empty group entry means that the rate limit group for that operation has not yet been activated.

Operation ID | Rate limit Group
-- | --
DeleteCharactersCharacterIdContacts | char-social
DeleteCharactersCharacterIdFittingsFittingId | fitting
DeleteCharactersCharacterIdMailLabelsLabelId | char-social
DeleteCharactersCharacterIdMailMailId | char-social
DeleteFleetsFleetIdMembersMemberId | fleet
DeleteFleetsFleetIdSquadsSquadId | fleet
DeleteFleetsFleetIdWingsWingId | fleet
GetAlliances | -
GetAlliancesAllianceId | -
GetAlliancesAllianceIdContacts | alliance-social
GetAlliancesAllianceIdContactsLabels | alliance-social
GetAlliancesAllianceIdCorporations | -
GetAlliancesAllianceIdIcons | -
GetCharactersCharacterId | -
GetCharactersCharacterIdAgentsResearch | char-industry
GetCharactersCharacterIdAssets | -
GetCharactersCharacterIdAttributes | char-detail
GetCharactersCharacterIdBlueprints | char-industry
GetCharactersCharacterIdCalendar | char-social
GetCharactersCharacterIdCalendarEventId | char-social
GetCharactersCharacterIdCalendarEventIdAttendees | char-social
GetCharactersCharacterIdClones | char-location
GetCharactersCharacterIdContacts | char-social
GetCharactersCharacterIdContactsLabels | char-social
GetCharactersCharacterIdContracts | char-contract
GetCharactersCharacterIdContractsContractIdBids | char-contract
GetCharactersCharacterIdContractsContractIdItems | char-contract
GetCharactersCharacterIdCorporationhistory | -
GetCharactersCharacterIdFatigue | char-location
GetCharactersCharacterIdFittings | fitting
GetCharactersCharacterIdFleet | fleet
GetCharactersCharacterIdFwStats | factional-warfare
GetCharactersCharacterIdImplants | char-detail
GetCharactersCharacterIdIndustryJobs | char-industry
GetCharactersCharacterIdKillmailsRecent | char-killmail
GetCharactersCharacterIdLocation | char-location
GetCharactersCharacterIdLoyaltyPoints | char-wallet
GetCharactersCharacterIdMail | char-social
GetCharactersCharacterIdMailLabels | char-social
GetCharactersCharacterIdMailLists | char-social
GetCharactersCharacterIdMailMailId | char-social
GetCharactersCharacterIdMedals | char-detail
GetCharactersCharacterIdMining | char-industry
GetCharactersCharacterIdNotifications | char-notification
GetCharactersCharacterIdNotificationsContacts | char-social
GetCharactersCharacterIdOnline | char-location
GetCharactersCharacterIdOrders | -
GetCharactersCharacterIdOrdersHistory | -
GetCharactersCharacterIdPlanets | char-industry
GetCharactersCharacterIdPlanetsPlanetId | char-industry
GetCharactersCharacterIdPortrait | char-detail
GetCharactersCharacterIdRoles | char-detail
GetCharactersCharacterIdSearch | -
GetCharactersCharacterIdShip | char-location
GetCharactersCharacterIdSkillqueue | char-detail
GetCharactersCharacterIdSkills | char-detail
GetCharactersCharacterIdStandings | char-social
GetCharactersCharacterIdTitles | char-detail
GetCharactersCharacterIdWallet | char-wallet
GetCharactersCharacterIdWalletJournal | char-wallet
GetCharactersCharacterIdWalletTransactions | char-wallet
GetContractsPublicBidsContractId | -
GetContractsPublicItemsContractId | -
GetContractsPublicRegionId | -
GetCorporationCorporationIdMiningExtractions | corp-industry
GetCorporationCorporationIdMiningObservers | corp-industry
GetCorporationCorporationIdMiningObserversObserverId | corp-industry
GetCorporationsCorporationId | -
GetCorporationsCorporationIdAlliancehistory | -
GetCorporationsCorporationIdAssets | -
GetCorporationsCorporationIdBlueprints | corp-industry
GetCorporationsCorporationIdContacts | corp-social
GetCorporationsCorporationIdContactsLabels | corp-social
GetCorporationsCorporationIdContainersLogs | -
GetCorporationsCorporationIdContracts | corp-contract
GetCorporationsCorporationIdContractsContractIdBids | corp-contract
GetCorporationsCorporationIdContractsContractIdItems | corp-contract
GetCorporationsCorporationIdCustomsOffices | corp-industry
GetCorporationsCorporationIdDivisions | corp-wallet
GetCorporationsCorporationIdFacilities | -
GetCorporationsCorporationIdFwStats | factional-warfare
GetCorporationsCorporationIdIcons | -
GetCorporationsCorporationIdIndustryJobs | corp-industry
GetCorporationsCorporationIdKillmailsRecent | corp-killmail
GetCorporationsCorporationIdMedals | corp-detail
GetCorporationsCorporationIdMedalsIssued | corp-detail
GetCorporationsCorporationIdMembers | corp-member
GetCorporationsCorporationIdMembersLimit | corp-member
GetCorporationsCorporationIdMembersTitles | corp-member
GetCorporationsCorporationIdMembertracking | corp-member
GetCorporationsCorporationIdOrders | -
GetCorporationsCorporationIdOrdersHistory | -
GetCorporationsCorporationIdRoles | corp-member
GetCorporationsCorporationIdRolesHistory | corp-member
GetCorporationsCorporationIdShareholders | corp-detail
GetCorporationsCorporationIdStandings | corp-member
GetCorporationsCorporationIdStarbases | -
GetCorporationsCorporationIdStarbasesStarbaseId | -
GetCorporationsCorporationIdStructures | -
GetCorporationsCorporationIdTitles | corp-detail
GetCorporationsCorporationIdWallets | corp-wallet
GetCorporationsCorporationIdWalletsDivisionJournal | corp-wallet
GetCorporationsCorporationIdWalletsDivisionTransactions | corp-wallet
GetCorporationsNpccorps | -
GetCorporationsProjectsContribution | -
GetCorporationsProjectsContributors | -
GetCorporationsProjectsDetail | -
GetCorporationsProjectsListing | -
GetDogmaAttributes | -
GetDogmaAttributesAttributeId | -
GetDogmaDynamicItemsTypeIdItemId | -
GetDogmaEffects | -
GetDogmaEffectsEffectId | -
GetFleetsFleetId | fleet
GetFleetsFleetIdMembers | fleet
GetFleetsFleetIdWings | fleet
GetFwLeaderboards | factional-warfare
GetFwLeaderboardsCharacters | factional-warfare
GetFwLeaderboardsCorporations | factional-warfare
GetFwStats | factional-warfare
GetFwSystems | factional-warfare
GetFwWars | factional-warfare
GetIncursions | incursion
GetIndustryFacilities | industry
GetIndustrySystems | industry
GetInsurancePrices | insurance
GetKillmailsKillmailIdKillmailHash | killmail
GetLoyaltyStoresCorporationIdOffers | -
GetMarketsGroups | -
GetMarketsGroupsMarketGroupId | -
GetMarketsPrices | -
GetMarketsRegionIdHistory | -
GetMarketsRegionIdOrders | -
GetMarketsRegionIdTypes | -
GetMarketsStructuresStructureId | -
GetMetaChangelog | -
GetMetaCompatibilityDates | -
GetMetaStatus | -
GetSovereigntyCampaigns | sovereignty
GetSovereigntyMap | sovereignty
GetSovereigntyStructures | sovereignty
GetStatus | status
GetUniverseAncestries | -
GetUniverseAsteroidBeltsAsteroidBeltId | -
GetUniverseBloodlines | -
GetUniverseCategories | -
GetUniverseCategoriesCategoryId | -
GetUniverseConstellations | -
GetUniverseConstellationsConstellationId | -
GetUniverseFactions | -
GetUniverseGraphics | -
GetUniverseGraphicsGraphicId | -
GetUniverseGroups | -
GetUniverseGroupsGroupId | -
GetUniverseMoonsMoonId | -
GetUniversePlanetsPlanetId | -
GetUniverseRaces | -
GetUniverseRegions | -
GetUniverseRegionsRegionId | -
GetUniverseSchematicsSchematicId | -
GetUniverseStargatesStargateId | -
GetUniverseStarsStarId | -
GetUniverseStationsStationId | -
GetUniverseStructures | -
GetUniverseStructuresStructureId | -
GetUniverseSystemJumps | -
GetUniverseSystemKills | -
GetUniverseSystems | -
GetUniverseSystemsSystemId | -
GetUniverseTypes | -
GetUniverseTypesTypeId | -
GetWars | killmail
GetWarsWarId | killmail
GetWarsWarIdKillmails | killmail
PostCharactersAffiliation | -
PostCharactersCharacterIdAssetsLocations | -
PostCharactersCharacterIdAssetsNames | -
PostCharactersCharacterIdContacts | char-social
PostCharactersCharacterIdCspa | char-detail
PostCharactersCharacterIdFittings | fitting
PostCharactersCharacterIdMail | char-social
PostCharactersCharacterIdMailLabels | char-social
PostCorporationsCorporationIdAssetsLocations | -
PostCorporationsCorporationIdAssetsNames | -
PostFleetsFleetIdMembers | fleet
PostFleetsFleetIdWings | fleet
PostFleetsFleetIdWingsWingIdSquads | fleet
PostRoute | routes
PostUiAutopilotWaypoint | ui
PostUiOpenwindowContract | ui
PostUiOpenwindowInformation | ui
PostUiOpenwindowMarketdetails | ui
PostUiOpenwindowNewmail | ui
PostUniverseIds | -
PostUniverseNames | -
PutCharactersCharacterIdCalendarEventId | char-social
PutCharactersCharacterIdContacts | char-social
PutCharactersCharacterIdMailMailId | char-social
PutFleetsFleetId | fleet
PutFleetsFleetIdMembersMemberId | fleet
PutFleetsFleetIdSquadsSquadId | fleet
PutFleetsFleetIdWingsWingId | fleet
