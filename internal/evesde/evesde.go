// Package evesde provides data from the EVE Online static data exchange.
package evesde

// NPCCorporationFactionID returns the faction ID for NPC corporation
// and reports whether the corporation was found.
func NPCCorporationFactionID(corporationID int64) (int64, bool) {
	id, ok := corporationToFactionID[corporationID]
	if !ok {
		return 0, false
	}
	return id, true
}
