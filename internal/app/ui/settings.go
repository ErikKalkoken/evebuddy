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
	settingMaxMails                           = "settingMaxMails"
	settingMaxMailsDefault                    = 1_000
	settingMaxMailsMax                        = 10_000
	settingMaxWalletTransactions              = "settingMaxWalletTransactions"
	settingMaxWalletTransactionsDefault       = 1_000
	settingMaxWalletTransactionsMax           = 10_000
	settingNotificationsTypesEnabled          = "settingNotificationsTypesEnabled"
	settingNotifyCommunicationsEarliest       = "settingNotifyCommunicationsEarliest"
	settingNotifyCommunicationsEnabled        = "settingNotifyCommunicationsEnabled"
	settingNotifyCommunicationsEnabledDefault = false
	settingNotifyContractsEarliest            = "settingNotifyContractsEarliest"
	settingNotifyContractsEnabled             = "settingNotifyContractsEnabled"
	settingNotifyContractsEnabledDefault      = false
	settingNotifyMailsEarliest                = "settingNotifyMailsEarliest"
	settingNotifyMailsEnabled                 = "settingNotifyMailsEnabled"
	settingNotifyMailsEnabledDefault          = false
	settingNotifyPIEarliest                   = "settingNotifyPIEarliest"
	settingNotifyPIEnabled                    = "settingNotifyPIEnabled"
	settingNotifyPIEnabledDefault             = false
	settingNotifyTimeoutHours                 = "settingNotifyTimeoutHours"
	settingNotifyTimeoutHoursDefault          = 30 * 24
	settingNotifyTimeoutHoursMax              = 90 * 24
	settingNotifyTrainingEarliest             = "settingNotifyTrainingEarliest"
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

// SettingKeys returns all setting keys. Mostly to know what to delete.
func SettingKeys() []string {
	return []string{
		settingLastCharacterID,
		settingMaxMails,
		settingMaxWalletTransactions,
		settingNotificationsTypesEnabled,
		settingNotifyCommunicationsEnabled,
		settingNotifyCommunicationsEarliest,
		settingNotifyContractsEnabled,
		settingNotifyContractsEarliest,
		settingNotifyMailsEnabled,
		settingNotifyMailsEarliest,
		settingNotifyPIEnabled,
		settingNotifyPIEarliest,
		settingNotifyTimeoutHours,
		settingNotifyTrainingEnabled,
		settingNotifyTrainingEarliest,
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
