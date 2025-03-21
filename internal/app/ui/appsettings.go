package ui

import (
	"log/slog"
	"maps"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

const (
	settingLogLevel                           = "logLevel"
	settingLogLevelDefault                    = "info"
	settingDeveloperMode                      = "developer-mode"
	settingDeveloperModeDefault               = false
	settingLastCharacterID                    = "settingLastCharacterID"
	settingMaxMails                           = "settingMaxMails"
	settingMaxMailsDefault                    = 1_000
	settingMaxMailsMax                        = 10_000
	settingMaxWalletTransactions              = "settingMaxWalletTransactions"
	settingMaxWalletTransactionsDefault       = 1_000
	settingMaxWalletTransactionsMax           = 10_000
	settingNotificationTypesEnabled           = "settingNotificationsTypesEnabled"
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
	settingNotifyTimeoutHoursMin              = 1
	settingNotifyTrainingEarliest             = "settingNotifyTrainingEarliest"
	settingNotifyTrainingEnabled              = "settingNotifyTrainingEnabled"
	settingNotifyTrainingEnabledDefault       = false
	settingSysTrayEnabled                     = "settingSysTrayEnabled"
	settingSysTrayEnabledDefault              = false
	settingTabsMainID                         = "tabs-main-id"
	settingTabsMainIDDefault                  = -1
	settingWindowHeightDefault                = 600
	settingWindowsSize                        = "window-size"
	settingWindowWidthDefault                 = 1000
)

type AppSettings struct {
	p fyne.Preferences
}

var _ app.Settings = (*AppSettings)(nil)

func NewAppSettings(p fyne.Preferences) *AppSettings {
	x := &AppSettings{p: p}
	return x
}

func (s AppSettings) DeveloperMode() bool {
	return s.p.BoolWithFallback(settingDeveloperMode, settingDeveloperModeDefault)
}

func (s AppSettings) ResetDeveloperMode() {
	s.SetDeveloperMode(settingDeveloperModeDefault)
}

func (s AppSettings) SetDeveloperMode(v bool) {
	s.p.SetBool(settingDeveloperMode, v)
}

func (s AppSettings) LogLevelNames() []string {
	x := slices.Collect(maps.Keys(logLevelName2Level))
	slices.Sort(x)
	return x
}

func (s AppSettings) LogLevelSlog() slog.Level {
	x := s.LogLevel()
	l, ok := logLevelName2Level[x]
	if !ok {
		l = logLevelName2Level[settingLogLevelDefault]
	}
	return l
}

var logLevelName2Level = map[string]slog.Level{
	"debug":   slog.LevelDebug,
	"error":   slog.LevelError,
	"info":    slog.LevelInfo,
	"warning": slog.LevelWarn,
}

func (s AppSettings) LogLevel() string {
	return s.p.StringWithFallback(settingLogLevel, settingLogLevelDefault)
}

func (s AppSettings) LogLevelDefault() string {
	return settingLogLevelDefault
}

func (s AppSettings) ResetLogLevel() {
	s.SetLogLevel(settingLogLevelDefault)
}

func (s AppSettings) SetLogLevel(l string) {
	s.p.SetString(settingLogLevel, l)
}

func (s AppSettings) MaxMails() int {
	return s.p.IntWithFallback(settingMaxMails, settingMaxMailsDefault)
}

func (s AppSettings) MaxMailsPresets() (min int, max int, def int) {
	min = 0
	max = settingMaxMailsMax
	def = settingMaxMailsDefault
	return
}

func (s AppSettings) ResetMaxMails() {
	s.SetMaxMails(settingMaxMailsDefault)
}

func (s AppSettings) SetMaxMails(v int) {
	s.p.SetInt(settingMaxMails, v)
}

func (s AppSettings) SysTrayEnabled() bool {
	return s.p.BoolWithFallback(settingSysTrayEnabled, settingSysTrayEnabledDefault)
}
func (s AppSettings) ResetSysTrayEnabled() {
	s.SetSysTrayEnabled(settingSysTrayEnabledDefault)
}

func (s AppSettings) SetSysTrayEnabled(v bool) {
	s.p.SetBool(settingSysTrayEnabled, v)
}

func (s AppSettings) WindowSize() fyne.Size {
	x := s.p.FloatList(settingWindowsSize)
	if len(x) < 2 {
		return fyne.NewSize(settingWindowWidthDefault, settingWindowHeightDefault)
	}
	return fyne.NewSize(float32(x[0]), float32(x[1]))
}

func (s AppSettings) ResetWindowSize() {
	s.SetWindowSize(fyne.NewSize(settingWindowWidthDefault, settingWindowHeightDefault))
}

func (s AppSettings) SetWindowSize(v fyne.Size) {
	s.p.SetFloatList(settingWindowsSize, []float64{float64(v.Width), float64(v.Height)})
}

func (s AppSettings) TabsMainID() int {
	return s.p.IntWithFallback(settingTabsMainID, settingTabsMainIDDefault)
}

func (s AppSettings) ResetTabsMainID() {
	s.SetTabsMainID(settingTabsMainIDDefault)
}

func (s AppSettings) SetTabsMainID(v int) {
	s.p.SetInt(settingTabsMainID, v)
}

func (s AppSettings) LastCharacterID() int32 {
	return int32(s.p.Int(settingLastCharacterID))
}

func (s AppSettings) ResetLastCharacterID() {
	s.SetLastCharacterID(0)
}

func (s AppSettings) SetLastCharacterID(id int32) {
	s.p.SetInt(settingLastCharacterID, int(id))
}

func (s AppSettings) MaxWalletTransactions() int {
	return s.p.IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
}

func (s AppSettings) MaxWalletTransactionsPresets() (min int, max int, def int) {
	min = 0
	max = settingMaxWalletTransactionsMax
	def = settingMaxWalletTransactionsDefault
	return
}

func (s AppSettings) ResetMaxWalletTransactions() {
	s.SetMaxWalletTransactions(settingMaxWalletTransactionsDefault)
}

func (s AppSettings) SetMaxWalletTransactions(v int) {
	s.p.SetInt(settingMaxWalletTransactions, v)
}

func (s AppSettings) NotifyTimeoutHours() int {
	return s.p.IntWithFallback(settingNotifyTimeoutHours, settingNotifyTimeoutHoursDefault)
}

func (s AppSettings) NotifyTimeoutHoursPresets() (min int, max int, def int) {
	min = settingNotifyTimeoutHoursMin
	max = settingNotifyTimeoutHoursMax
	def = settingNotifyTimeoutHoursDefault
	return
}

func (s AppSettings) ResetNotifyTimeoutHours() {
	s.SetNotifyTimeoutHours(settingNotifyTimeoutHoursDefault)
}

func (s AppSettings) SetNotifyTimeoutHours(v int) {
	s.p.SetInt(settingNotifyTimeoutHours, v)
}

func (s AppSettings) NotificationTypesEnabled() set.Set[string] {
	return set.NewFromSlice(s.p.StringList(settingNotificationTypesEnabled))
}

func (s AppSettings) ResetNotificationTypesEnabled() {
	s.SetNotificationTypesEnabled(set.New[string]())
}

func (s AppSettings) SetNotificationTypesEnabled(v set.Set[string]) {
	s.p.SetStringList(settingNotificationTypesEnabled, v.ToSlice())
}

func (s AppSettings) NotifyCommunicationsEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyCommunicationsEarliest)
}

func (s AppSettings) SetNotifyCommunicationsEarliest(t time.Time) {
	s.setEarliest(settingNotifyCommunicationsEarliest, t)
}

func (s AppSettings) NotifyContractsEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyContractsEarliest)
}

func (s AppSettings) SetNotifyContractsEarliest(t time.Time) {
	s.setEarliest(settingNotifyContractsEarliest, t)
}

func (s AppSettings) NotifyMailsEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyMailsEarliest)
}
func (s AppSettings) SetNotifyMailsEarliest(t time.Time) {
	s.setEarliest(settingNotifyMailsEarliest, t)
}

func (s AppSettings) NotifyPIEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyPIEarliest)
}
func (s AppSettings) SetNotifyPIEarliest(t time.Time) {
	s.setEarliest(settingNotifyPIEarliest, t)
}

func (s AppSettings) NotifyTrainingEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyTrainingEarliest)
}
func (s AppSettings) SetNotifyTrainingEarliest(t time.Time) {
	s.setEarliest(settingNotifyTrainingEarliest, t)
}

// func (s AppSettings) getEarliest(key string) time.Time {
// 	x := s.p.String(key)
// 	t, ok := string2time(x)
// 	if !ok {
// 		// Recording the earliest when enabling a switch was added later for mails and communications
// 		// This workaround avoids a potential notification spam from older items.
// 		t = time.Now().UTC().Add(-notifyEarliestFallback)
// 		s.setEarliest(key, t)
// 	}
// 	return t
// }

func (s AppSettings) setEarliest(key string, t time.Time) {
	s.p.SetString(key, timeToString(t))
}

// calcNotifyEarliest returns the earliest time for a class of notifications.
// Might return a zero time in some circumstances.
func (s AppSettings) calcNotifyEarliest(key string) time.Time {
	earliest, ok := string2time(s.p.String(key))
	if !ok {
		// Recording the earliest when enabling a switch was added later for mails and communications
		// This workaround avoids a potential notification spam from older items.
		earliest = time.Now().UTC().Add(-notifyEarliestFallback)
		s.setEarliest(key, earliest)
	}
	timeoutHours := s.NotifyTimeoutHours()
	var timeout time.Time
	if timeoutHours > 0 {
		timeout = time.Now().UTC().Add(-time.Duration(timeoutHours) * time.Hour)
	}
	if earliest.After(timeout) {
		return earliest
	}
	return timeout
}

func timeToString(t time.Time) string {
	return t.Format(time.RFC3339)
}

func string2time(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		slog.Error("string2time", "string", s, "error", err)
		return time.Time{}, false
	}
	return t, true
}

func (s AppSettings) NotifyCommunicationsEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault)
}
func (s AppSettings) ResetNotifyCommunicationsEnabled() {
	s.SetNotifyCommunicationsEnabled(settingNotifyCommunicationsEnabledDefault)
}

func (s AppSettings) SetNotifyCommunicationsEnabled(v bool) {
	s.p.SetBool(settingNotifyCommunicationsEnabled, v)
}

func (s AppSettings) NotifyContractsEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyContractsEnabled, settingNotifyContractsEnabledDefault)
}
func (s AppSettings) ResetNotifyContractsEnabled() {
	s.SetNotifyContractsEnabled(settingNotifyContractsEnabledDefault)
}

func (s AppSettings) SetNotifyContractsEnabled(v bool) {
	s.p.SetBool(settingNotifyContractsEnabled, v)
}

func (s AppSettings) NotifyMailsEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault)
}
func (s AppSettings) ResetNotifyMailsEnabled() {
	s.SetNotifyMailsEnabled(settingNotifyMailsEnabledDefault)
}

func (s AppSettings) SetNotifyMailsEnabled(v bool) {
	s.p.SetBool(settingNotifyMailsEnabled, v)
}

func (s AppSettings) NotifyPIEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyPIEnabled, settingNotifyPIEnabledDefault)
}
func (s AppSettings) ResetNotifyPIEnabled() {
	s.SetNotifyPIEnabled(settingNotifyPIEnabledDefault)
}

func (s AppSettings) SetNotifyPIEnabled(v bool) {
	s.p.SetBool(settingNotifyPIEnabled, v)
}

func (s AppSettings) NotifyTrainingEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault)
}
func (s AppSettings) ResetNotifyTrainingEnabled() {
	s.SetNotifyTrainingEnabled(settingNotifyTrainingEnabledDefault)
}

func (s AppSettings) SetNotifyTrainingEnabled(v bool) {
	s.p.SetBool(settingNotifyTrainingEnabled, v)
}

// SettingKeys returns all setting keys. Mostly to know what to delete.
func SettingKeys() []string {
	return []string{
		settingLastCharacterID,
		settingMaxMails,
		settingMaxWalletTransactions,
		settingNotificationTypesEnabled,
		settingNotifyCommunicationsEarliest,
		settingNotifyCommunicationsEnabled,
		settingNotifyContractsEarliest,
		settingNotifyContractsEnabled,
		settingNotifyMailsEarliest,
		settingNotifyMailsEnabled,
		settingNotifyPIEarliest,
		settingNotifyPIEnabled,
		settingNotifyTimeoutHours,
		settingNotifyTrainingEarliest,
		settingNotifyTrainingEnabled,
		settingSysTrayEnabled,
		settingTabsMainID,
		settingWindowsSize,
	}
}
