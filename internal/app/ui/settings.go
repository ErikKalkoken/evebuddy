package ui

import (
	"log/slog"
	"maps"
	"slices"
)

// Settings
const (
	settingLastCharacterID                    = "settingLastCharacterID"
	SettingLogLevel                           = "logLevel"
	SettingLogLevelDefault                    = "warning"
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
	settingNotifyContractsEnabled             = "settingNotifyContractsEnabled"
	settingNotifyContractsEnabledDefault      = false
	settingNotifyContractsEarliest            = "settingNotifyContractsEarliest"
	settingNotifyMailsEnabled                 = "settingNotifyMailsEnabled"
	settingNotifyMailsEnabledDefault          = false
	settingNotifyPIEarliest                   = "settingNotifyPIEarliest"
	settingNotifyPIEnabled                    = "settingNotifyPIEnabled"
	settingNotifyPIEnabledDefault             = false
	settingNotifyTrainingEnabled              = "settingNotifyTrainingEnabled"
	settingNotifyTrainingEnabledDefault       = false
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
		settingNotifyContractsEnabled,
		settingNotifyMailsEnabled,
		settingNotifyPIEnabled,
		settingNotifyTrainingEnabled,
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
