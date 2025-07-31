// Package settings provides an API for reading and writing the app's settings.
package settings

import (
	"log/slog"
	"maps"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type ColorTheme string

const (
	Auto  ColorTheme = "Auto"
	Light ColorTheme = "Light"
	Dark  ColorTheme = "Dark"
)

const (
	notifyEarliestFallback                    = 24 * time.Hour
	settingColorTheme                         = "color-theme"
	settingColorThemeDefault                  = Auto
	settingDeveloperMode                      = "developer-mode"
	settingDeveloperModeDefault               = false
	settingLastCharacterID                    = "settingLastCharacterID"
	settingLastCorporationID                  = "settingLastCorporationID"
	settingLogLevel                           = "logLevel"
	settingLogLevelDefault                    = "info"
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
	settingPreferMarketTab                    = "settingPreferMarketTab"
	settingRecentSearches                     = "settingRecentSearches"
	settingSysTrayEnabled                     = "settingSysTrayEnabled"
	settingSysTrayEnabledDefault              = true
	settingTabsMainID                         = "tabs-main-id"
	settingTabsMainIDDefault                  = -1
	settingFyneDisableDPIDetection            = "settingFyneDisableDPIDetection"
	settingFyneScale                          = "settingFyneScale"
	settingFyneScaleDefault                   = 1.0
	settingWindowHeightDefault                = 600
	settingWindowsSize                        = "window-size"
	settingWindowWidthDefault                 = 1000
)

// Settings represents the settings for the app and provides an API for reading and writing settings.
type Settings struct {
	p fyne.Preferences
}

// New returns a new Settings object.
func New(p fyne.Preferences) *Settings {
	x := &Settings{p: p}
	return x
}

func (s Settings) DeveloperMode() bool {
	return s.p.BoolWithFallback(settingDeveloperMode, settingDeveloperModeDefault)
}

func (s Settings) ResetDeveloperMode() {
	s.SetDeveloperMode(settingDeveloperModeDefault)
}

func (s Settings) SetDeveloperMode(v bool) {
	s.p.SetBool(settingDeveloperMode, v)
}

func (s Settings) LogLevelNames() []string {
	x := slices.Collect(maps.Keys(logLevelName2Level))
	slices.Sort(x)
	return x
}

func (s Settings) LogLevelSlog() slog.Level {
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

func (s Settings) LogLevel() string {
	return s.p.StringWithFallback(settingLogLevel, settingLogLevelDefault)
}

func (s Settings) LogLevelDefault() string {
	return settingLogLevelDefault
}

func (s Settings) ResetLogLevel() {
	s.SetLogLevel(settingLogLevelDefault)
}

func (s Settings) SetLogLevel(l string) {
	s.p.SetString(settingLogLevel, l)
}

func (s Settings) MaxMails() int {
	return s.p.IntWithFallback(settingMaxMails, settingMaxMailsDefault)
}

func (s Settings) MaxMailsPresets() (min int, max int, def int) {
	min = 0
	max = settingMaxMailsMax
	def = settingMaxMailsDefault
	return
}

func (s Settings) ResetMaxMails() {
	s.SetMaxMails(settingMaxMailsDefault)
}

func (s Settings) SetMaxMails(v int) {
	s.p.SetInt(settingMaxMails, v)
}

func (s Settings) SysTrayEnabled() bool {
	return s.p.BoolWithFallback(settingSysTrayEnabled, settingSysTrayEnabledDefault)
}
func (s Settings) ResetSysTrayEnabled() {
	s.SetSysTrayEnabled(settingSysTrayEnabledDefault)
}

func (s Settings) SetSysTrayEnabled(v bool) {
	s.p.SetBool(settingSysTrayEnabled, v)
}

func (s Settings) WindowSize() fyne.Size {
	x := s.p.FloatList(settingWindowsSize)
	if len(x) < 2 {
		return fyne.NewSize(settingWindowWidthDefault, settingWindowHeightDefault)
	}
	return fyne.NewSize(float32(x[0]), float32(x[1]))
}

func (s Settings) ResetWindowSize() {
	s.SetWindowSize(fyne.NewSize(settingWindowWidthDefault, settingWindowHeightDefault))
}

func (s Settings) SetWindowSize(v fyne.Size) {
	s.p.SetFloatList(settingWindowsSize, []float64{float64(v.Width), float64(v.Height)})
}

func (s Settings) ResetTabsMainID() {
	s.SetTabsMainID(settingTabsMainIDDefault)
}

func (s Settings) SetTabsMainID(v int) {
	s.p.SetInt(settingTabsMainID, v)
}

func (s Settings) LastCharacterID() int32 {
	return int32(s.p.Int(settingLastCharacterID))
}

func (s Settings) ResetLastCharacterID() {
	s.SetLastCharacterID(0)
}

func (s Settings) SetLastCharacterID(id int32) {
	s.p.SetInt(settingLastCharacterID, int(id))
}

func (s Settings) LastCorporationID() int32 {
	return int32(s.p.Int(settingLastCorporationID))
}

func (s Settings) ResetLastCorporationID() {
	s.SetLastCorporationID(0)
}

func (s Settings) SetLastCorporationID(id int32) {
	s.p.SetInt(settingLastCorporationID, int(id))
}

func (s Settings) MaxWalletTransactions() int {
	return s.p.IntWithFallback(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
}

func (s Settings) MaxWalletTransactionsPresets() (min int, max int, def int) {
	min = 0
	max = settingMaxWalletTransactionsMax
	def = settingMaxWalletTransactionsDefault
	return
}

func (s Settings) ResetMaxWalletTransactions() {
	s.SetMaxWalletTransactions(settingMaxWalletTransactionsDefault)
}

func (s Settings) SetMaxWalletTransactions(v int) {
	s.p.SetInt(settingMaxWalletTransactions, v)
}

func (s Settings) NotifyTimeoutHours() int {
	return s.p.IntWithFallback(settingNotifyTimeoutHours, settingNotifyTimeoutHoursDefault)
}

func (s Settings) NotifyTimeoutHoursPresets() (min int, max int, def int) {
	min = settingNotifyTimeoutHoursMin
	max = settingNotifyTimeoutHoursMax
	def = settingNotifyTimeoutHoursDefault
	return
}

func (s Settings) ResetNotifyTimeoutHours() {
	s.SetNotifyTimeoutHours(settingNotifyTimeoutHoursDefault)
}

func (s Settings) SetNotifyTimeoutHours(v int) {
	s.p.SetInt(settingNotifyTimeoutHours, v)
}

func (s Settings) NotificationTypesEnabled() set.Set[string] {
	return set.Of(s.p.StringList(settingNotificationTypesEnabled)...)
}

func (s Settings) ResetNotificationTypesEnabled() {
	s.SetNotificationTypesEnabled(set.Of[string]())
}

func (s Settings) SetNotificationTypesEnabled(v set.Set[string]) {
	s.p.SetStringList(settingNotificationTypesEnabled, v.Slice())
}

func (s Settings) NotifyCommunicationsEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyCommunicationsEarliest)
}

func (s Settings) SetNotifyCommunicationsEarliest(t time.Time) {
	s.setEarliest(settingNotifyCommunicationsEarliest, t)
}

func (s Settings) NotifyContractsEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyContractsEarliest)
}

func (s Settings) SetNotifyContractsEarliest(t time.Time) {
	s.setEarliest(settingNotifyContractsEarliest, t)
}

func (s Settings) NotifyMailsEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyMailsEarliest)
}
func (s Settings) SetNotifyMailsEarliest(t time.Time) {
	s.setEarliest(settingNotifyMailsEarliest, t)
}

func (s Settings) NotifyPIEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyPIEarliest)
}
func (s Settings) SetNotifyPIEarliest(t time.Time) {
	s.setEarliest(settingNotifyPIEarliest, t)
}

func (s Settings) NotifyTrainingEarliest() time.Time {
	return s.calcNotifyEarliest(settingNotifyTrainingEarliest)
}
func (s Settings) SetNotifyTrainingEarliest(t time.Time) {
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

func (s Settings) setEarliest(key string, t time.Time) {
	s.p.SetString(key, timeToString(t))
}

// calcNotifyEarliest returns the earliest time for a class of notifications.
// Might return a zero time in some circumstances.
func (s Settings) calcNotifyEarliest(key string) time.Time {
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

func (s Settings) NotifyCommunicationsEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault)
}
func (s Settings) ResetNotifyCommunicationsEnabled() {
	s.SetNotifyCommunicationsEnabled(settingNotifyCommunicationsEnabledDefault)
}

func (s Settings) SetNotifyCommunicationsEnabled(v bool) {
	s.p.SetBool(settingNotifyCommunicationsEnabled, v)
}

func (s Settings) NotifyContractsEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyContractsEnabled, settingNotifyContractsEnabledDefault)
}
func (s Settings) ResetNotifyContractsEnabled() {
	s.SetNotifyContractsEnabled(settingNotifyContractsEnabledDefault)
}

func (s Settings) SetNotifyContractsEnabled(v bool) {
	s.p.SetBool(settingNotifyContractsEnabled, v)
}

func (s Settings) NotifyMailsEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault)
}
func (s Settings) ResetNotifyMailsEnabled() {
	s.SetNotifyMailsEnabled(settingNotifyMailsEnabledDefault)
}

func (s Settings) SetNotifyMailsEnabled(v bool) {
	s.p.SetBool(settingNotifyMailsEnabled, v)
}

func (s Settings) NotifyPIEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyPIEnabled, settingNotifyPIEnabledDefault)
}
func (s Settings) ResetNotifyPIEnabled() {
	s.SetNotifyPIEnabled(settingNotifyPIEnabledDefault)
}

func (s Settings) SetNotifyPIEnabled(v bool) {
	s.p.SetBool(settingNotifyPIEnabled, v)
}

func (s Settings) NotifyTrainingEnabled() bool {
	return s.p.BoolWithFallback(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault)
}

func (s Settings) ResetNotifyTrainingEnabled() {
	s.SetNotifyTrainingEnabled(settingNotifyTrainingEnabledDefault)
}

func (s Settings) SetNotifyTrainingEnabled(v bool) {
	s.p.SetBool(settingNotifyTrainingEnabled, v)
}

func (s Settings) TabsMainID() int {
	return s.p.IntWithFallback(settingTabsMainID, settingTabsMainIDDefault)
}

func (s Settings) RecentSearches() []int32 {
	return xslices.Map(s.p.IntList(settingRecentSearches), func(x int) int32 {
		return int32(x)
	})
}
func (s Settings) SetRecentSearches(v []int32) {
	s.p.SetIntList(settingRecentSearches, xslices.Map(v, func(x int32) int {
		return int(x)
	}))
}

func (s Settings) PreferMarketTab() bool {
	return s.p.Bool(settingPreferMarketTab)
}
func (s Settings) ResetPreferMarketTab() {
	s.SetPreferMarketTab(false)
}

func (s Settings) SetPreferMarketTab(v bool) {
	s.p.SetBool(settingPreferMarketTab, v)
}

func (s Settings) ColorTheme() ColorTheme {
	x := s.p.StringWithFallback(settingColorTheme, string(settingColorThemeDefault))
	return ColorTheme(x)
}

func (s Settings) ColorThemeDefault() ColorTheme {
	return settingColorThemeDefault
}

func (s Settings) ResetColorTheme() {
	s.SetColorTheme(settingColorThemeDefault)
}

func (s Settings) SetColorTheme(v ColorTheme) {
	s.p.SetString(settingColorTheme, string(v))
}

func (s Settings) FyneScale() float64 {
	return s.p.FloatWithFallback(settingFyneScale, settingFyneScaleDefault)
}

func (s Settings) FyneScaleDefault() float64 {
	return settingFyneScaleDefault
}

func (s Settings) ResetFyneScale() {
	s.SetFyneScale(settingFyneScaleDefault)
}

func (s Settings) SetFyneScale(v float64) {
	s.p.SetFloat(settingFyneScale, v)
}

func (s Settings) FyneDisableDPIDetection() bool {
	return s.p.Bool(settingFyneDisableDPIDetection)
}

func (s Settings) ResetSetFyneDisableDPIDetection() {
	s.SetFyneDisableDPIDetection(false)
}

func (s Settings) SetFyneDisableDPIDetection(v bool) {
	s.p.SetBool(settingFyneDisableDPIDetection, v)
}

// Keys returns all setting keys. Mostly to know what to delete.
func Keys() []string {
	return []string{
		settingDeveloperMode,
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
		settingRecentSearches,
		settingSysTrayEnabled,
		settingTabsMainID,
		settingWindowsSize,
	}
}
