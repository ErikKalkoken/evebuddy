package app

import (
	"maps"
	"slices"
)

type NotificationGroup uint

const (
	GroupBills NotificationGroup = iota + 1
	GroupFactionWarfare
	GroupContacts
	GroupCorporate
	GroupInsurance
	GroupInsurgencies
	GroupMoonMining
	GroupMiscellaneous
	GroupOld
	GroupSovereignty
	GroupStructure
	GroupWar
	GroupUnknown
	GroupUnread
	GroupAll
)

func (c NotificationGroup) String() string {
	return group2Name[c]
}

var group2Name = map[NotificationGroup]string{
	GroupBills:          "Bills",
	GroupFactionWarfare: "Faction Warfare",
	GroupContacts:       "Contacts",
	GroupCorporate:      "Corporate",
	GroupInsurance:      "Insurance",
	GroupInsurgencies:   "Insurgencies",
	GroupMiscellaneous:  "Miscellaneous",
	GroupMoonMining:     "Moon Mining",
	GroupOld:            "Old",
	GroupSovereignty:    "Sovereignty",
	GroupStructure:      "Structure",
	GroupWar:            "War",
	GroupUnread:         "Unread",
	GroupUnknown:        "Unknown",
	GroupAll:            "All",
}

// Groups returns a slice of all groups in alphabetical order.
func NotificationGroups() []NotificationGroup {
	return slices.Sorted(maps.Keys(group2Name))
}
