package app

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
	GroupAll:            "All",
	GroupBills:          "Bills",
	GroupContacts:       "Contacts",
	GroupCorporate:      "Corporate",
	GroupFactionWarfare: "Faction Warfare",
	GroupInsurance:      "Insurance",
	GroupInsurgencies:   "Insurgencies",
	GroupMiscellaneous:  "Miscellaneous",
	GroupMoonMining:     "Moon Mining",
	GroupOld:            "Old",
	GroupSovereignty:    "Sovereignty",
	GroupStructure:      "Structure",
	GroupUnknown:        "Unknown",
	GroupUnread:         "Unread",
	GroupWar:            "War",
}

// NotificationGroups returns a slice of all normal groups in alphabetical order.
func NotificationGroups() []NotificationGroup {
	return []NotificationGroup{
		GroupBills,
		GroupFactionWarfare,
		GroupContacts,
		GroupCorporate,
		GroupInsurance,
		GroupInsurgencies,
		GroupMoonMining,
		GroupMiscellaneous,
		GroupOld,
		GroupSovereignty,
		GroupStructure,
		GroupWar,
	}
}
