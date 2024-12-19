package ui

import (
	"log/slog"
	"maps"
	"slices"
)

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
	settingNotifyPIEnabled                    = "settingNotifyPIEnabled"
	settingNotifyPIEnabledDefault             = false
	settingNotifyTrainingEnabled              = "settingNotifyTrainingEnabled"
	settingNotifyTrainingEnabledDefault       = false
	settingNotifyPIEarliest                   = "settingNotifyPIEarliest"
	settingSysTrayEnabled                     = "settingSysTrayEnabled"
	settingSysTrayEnabledDefault              = false
	settingTabsMainID                         = "tabs-main-id"
	settingWindowHeight                       = "window-height"
	settingWindowHeightDefault                = 600
	settingWindowWidth                        = "window-width"
	settingWindowWidthDefault                 = 1000
	SettingLogLevel                           = "logLevel"
	SettingLogLevelDefault                    = "warning"
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
		settingNotifyPIEnabled,
		settingSysTrayEnabled,
		settingTabsMainID,
		settingWindowHeight,
		settingWindowWidth,
	}
}

var logLevelName2Level = map[string]slog.Level{
	"debug":   slog.LevelDebug,
	"error":   slog.LevelError,
	"info":    slog.LevelInfo,
	"warning": slog.LevelWarn,
}

func LogLevelName2Level(s string) slog.Level {
	l, ok := logLevelName2Level[s]
	if !ok {
		l = slog.LevelInfo
	}
	return l
}

func LogLevelNames() []string {
	x := slices.Collect(maps.Keys(logLevelName2Level))
	slices.Sort(x)
	return x
}
