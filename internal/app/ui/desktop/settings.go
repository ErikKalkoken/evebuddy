package desktop

// Settings
const (
	settingSysTrayEnabled        = "settingSysTrayEnabled"
	settingSysTrayEnabledDefault = false
	settingTabsMainID            = "tabs-main-id"
	settingWindowHeight          = "window-height"
	settingWindowHeightDefault   = 600
	settingWindowWidth           = "window-width"
	settingWindowWidthDefault    = 1000
)

// SettingKeys returns all setting keys. Mostly to know what to delete.
func SettingKeys() []string {
	return []string{
		settingSysTrayEnabled,
		settingTabsMainID,
		settingWindowHeight,
		settingWindowWidth,
	}
}
