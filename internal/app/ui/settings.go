package ui

import (
	"log/slog"
	"maps"
	"slices"
)

const (
	SettingLastCharacterID                    = "settingLastCharacterID"
	SettingLogLevel                           = "logLevel"
	SettingLogLevelDefault                    = "warning"
	SettingMaxMails                           = "settingMaxMails"
	SettingMaxMailsDefault                    = 1_000
	SettingMaxMailsMax                        = 10_000
	SettingMaxWalletTransactions              = "settingMaxWalletTransactions"
	SettingMaxWalletTransactionsDefault       = 1_000
	SettingMaxWalletTransactionsMax           = 10_000
	SettingNotificationsTypesEnabled          = "settingNotificationsTypesEnabled"
	SettingNotifyCommunicationsEarliest       = "settingNotifyCommunicationsEarliest"
	SettingNotifyCommunicationsEnabled        = "settingNotifyCommunicationsEnabled"
	SettingNotifyCommunicationsEnabledDefault = false
	SettingNotifyContractsEarliest            = "settingNotifyContractsEarliest"
	SettingNotifyContractsEnabled             = "settingNotifyContractsEnabled"
	SettingNotifyContractsEnabledDefault      = false
	SettingNotifyMailsEarliest                = "settingNotifyMailsEarliest"
	SettingNotifyMailsEnabled                 = "settingNotifyMailsEnabled"
	SettingNotifyMailsEnabledDefault          = false
	SettingNotifyPIEarliest                   = "settingNotifyPIEarliest"
	SettingNotifyPIEnabled                    = "settingNotifyPIEnabled"
	SettingNotifyPIEnabledDefault             = false
	SettingNotifyTimeoutHours                 = "settingNotifyTimeoutHours"
	SettingNotifyTimeoutHoursDefault          = 30 * 24
	SettingNotifyTimeoutHoursMax              = 90 * 24
	SettingNotifyTrainingEarliest             = "settingNotifyTrainingEarliest"
	SettingNotifyTrainingEnabled              = "settingNotifyTrainingEnabled"
	SettingNotifyTrainingEnabledDefault       = false
)

// SettingKeys returns all setting keys. Mostly to know what to delete.
func SettingKeys() []string {
	return []string{
		SettingLastCharacterID,
		SettingMaxMails,
		SettingMaxWalletTransactions,
		SettingNotificationsTypesEnabled,
		SettingNotifyCommunicationsEnabled,
		SettingNotifyCommunicationsEarliest,
		SettingNotifyContractsEnabled,
		SettingNotifyContractsEarliest,
		SettingNotifyMailsEnabled,
		SettingNotifyMailsEarliest,
		SettingNotifyPIEnabled,
		SettingNotifyPIEarliest,
		SettingNotifyTimeoutHours,
		SettingNotifyTrainingEnabled,
		SettingNotifyTrainingEarliest,
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
