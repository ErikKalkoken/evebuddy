package ui

// Settings
const (
	settingLastCharacterID                    = "settingLastCharacterID"
	settingMaxAge                             = "settingMaxAgeHours"
	settingMaxAgeDefault                      = 6  // hours
	settingMaxAgeMax                          = 24 // hours
	settingMaxMails                           = "settingMaxMails"
	settingMaxMailsDefault                    = 1_000
	settingMaxMailsMax                        = 10_000
	settingMaxWalletTransactions              = "settingMaxWalletTransactions"
	settingMaxWalletTransactionsDefault       = 1_000
	settingMaxWalletTransactionsMax           = 10_000
	settingNotificationsTypesEnabled          = "settingNotificationsTypesEnabled"
	settingNotifyCommunicationsEnabled        = "settingNotifyCommunicationsEnabled"
	settingNotifyCommunicationsEnabledDefault = false
	settingNotifyMailsEnabled                 = "settingNotifyMailsEnabled"
	settingNotifyMailsEnabledDefault          = false
	settingSysTrayEnabled                     = "settingSysTrayEnabled"
	settingSysTrayEnabledDefault              = false
	settingTabsMainID                         = "tabs-main-id"
	settingWindowHeight                       = "window-height"
	settingWindowHeightDefault                = 600
	settingWindowWidth                        = "window-width"
	settingWindowWidthDefault                 = 1000
)

// SettingKeys returns all setting keys.
func SettingKeys() []string {
	return []string{
		settingLastCharacterID,
		settingMaxAge,
		settingMaxMails,
		settingMaxWalletTransactions,
		settingNotificationsTypesEnabled,
		settingNotifyCommunicationsEnabled,
		settingNotifyMailsEnabled,
		settingSysTrayEnabled,
		settingTabsMainID,
		settingWindowHeight,
		settingWindowWidth,
	}
}
