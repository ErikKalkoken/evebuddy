// Package settings provides an API for reading and writing the app's settings.
package settings

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/syncqueue"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type ColorTheme string

const (
	Auto  ColorTheme = "Auto"
	Light ColorTheme = "Light"
	Dark  ColorTheme = "Dark"
)

// Keys, defaults and presets for settings.
// Keys without default have the zero value as default.
const (
	notifyEarliestFallback                    = 24 * time.Hour
	settingColorTheme                         = "color-theme"
	settingColorThemeDefault                  = Auto
	settingDeveloperMode                      = "developer-mode"
	settingLastCharacterID                    = "settingLastCharacterID"
	settingLastCorporationID                  = "settingLastCorporationID"
	settingLogLevel                           = "logLevel"
	settingLogLevelDefault                    = "info"
	settingApprovedContactCost                = "settingApprovedContactCost"
	settingApprovedContactCostMax             = 1_000_000
	settingMaxMails                           = "settingMaxMails"
	settingMaxMailsDefault                    = 250
	settingMaxMailsMax                        = 10_000
	settingMaxWalletTransactions              = "settingMaxWalletTransactions"
	settingMarketOrdersRetentionDays          = "settingMarketOrderRetentionDays"
	settingMarketOrderRetentionDaysMin        = 30
	settingMarketOrderRetentionDaysDefault    = 90
	settingMarketOrderRetentionDaysMax        = 360
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
	settingDisableDPIDetection                = "settingFyneDisableDPIDetection"
	settingFyneScale                          = "settingFyneScale"
	settingFyneScaleDefault                   = 1.0
	settingWindowHeightDefault                = 600
	settingWindowsSize                        = "window-size"
	settingWindowWidthDefault                 = 1000
	settingHideLimitedCorporations            = "settingHideLimitedCorporations"
	settingHideLimitedCorporationsDefault     = false
)

// Settings represents the settings for the app and provides an API for reading and writing settings.
//
// Values are cached in memory and persisted to storage on every write. The whole
// table is preloaded once at construction time, so reads never touch storage.
type Settings struct {
	st  *storage.Storage
	ctx context.Context // held deliberately: keeps every public method free of a ctx param

	mu     sync.RWMutex
	values map[string]string

	writeQueue *syncqueue.SyncQueue[settingWrite]
}

// New returns a new Settings object, preloading all currently stored values.
func New(ctx context.Context, st *storage.Storage) (*Settings, error) {
	s := &Settings{
		st:         st,
		ctx:        ctx,
		values:     make(map[string]string),
		writeQueue: syncqueue.New[settingWrite](),
	}
	rows, err := st.ListSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("settings: load: %w", err)
	}
	for _, r := range rows {
		s.values[r.Key] = r.Value
	}
	go s.persistLoop()
	return s, nil
}

// persistLoop writes queued setting values to storage one at a time, off the
// caller's goroutine, so that Set* calls from a UI callback never block on disk I/O.
func (s *Settings) persistLoop() {
	for {
		w, err := s.writeQueue.Get(s.ctx)
		if err != nil {
			return // s.ctx was canceled
		}
		if w.done != nil {
			close(w.done)
			continue
		}
		if err := s.st.SetSetting(s.ctx, w.key, w.value); err != nil {
			slog.Error("settings: failed to persist value", "key", w.key, "error", err)
		}
	}
}

func (s *Settings) DeveloperMode() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingDeveloperMode, false)
}

func (s *Settings) SetDeveloperMode(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingDeveloperMode, v)
}

// LogLevelNames returns the names of all log levels in ascending order of severity.
func (s *Settings) LogLevelNames() []string {
	if s == nil {
		return nil
	}

	type t struct {
		name  string
		level slog.Level
	}
	var levels []t
	for name, level := range logLevelName2Level {
		levels = append(levels, t{name, level})
	}
	slices.SortFunc(levels, func(a, b t) int {
		return cmp.Compare(a.level, b.level)
	})
	names := xslices.Map(levels, func(x t) string {
		return x.name
	})
	return names
}

func (s *Settings) LogLevelSlog() slog.Level {
	if s == nil {
		return slog.Level(0)
	}
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

func (s *Settings) LogLevel() string {
	if s == nil {
		return ""
	}
	return s.getString(settingLogLevel, settingLogLevelDefault)
}

func (s *Settings) LogLevelDefault() string {
	if s == nil {
		return ""
	}
	return settingLogLevelDefault
}

func (s *Settings) ResetLogLevel() {
	if s == nil {
		return
	}
	s.SetLogLevel(settingLogLevelDefault)
}

func (s *Settings) SetLogLevel(l string) {
	if s == nil {
		return
	}
	s.setString(settingLogLevel, l)
}

func (s *Settings) ApprovedContactCost() int {
	if s == nil {
		return 0
	}
	return s.getInt(settingApprovedContactCost, 0)
}

func (s *Settings) ApprovedContactCostPresets() (minimum int, maximum int, def int) {
	minimum = 0
	maximum = settingApprovedContactCostMax
	def = 0
	return
}

func (s *Settings) SetApprovedContactCost(v int) {
	if s == nil {
		return
	}
	s.setInt(settingApprovedContactCost, v)
}

func (s *Settings) MaxMails() int {
	if s == nil {
		return 0
	}
	return s.getInt(settingMaxMails, settingMaxMailsDefault)
}

func (s *Settings) MaxMailsPresets() (minimum int, maximum int, def int) {
	minimum = 0
	maximum = settingMaxMailsMax
	def = settingMaxMailsDefault
	return
}

func (s *Settings) SetMaxMails(v int) {
	if s == nil {
		return
	}
	s.setInt(settingMaxMails, v)
}

func (s *Settings) MarketOrderRetentionDays() int {
	if s == nil {
		return 0
	}
	return s.getInt(settingMarketOrdersRetentionDays, settingMarketOrderRetentionDaysDefault)
}

func (s *Settings) MarketOrderRetentionDaysPresets() (minimum int, maximum int, def int) {
	minimum = settingMarketOrderRetentionDaysMin
	maximum = settingMarketOrderRetentionDaysMax
	def = settingMarketOrderRetentionDaysDefault
	return
}

func (s *Settings) SetMarketOrdersRetentionDay(v int) {
	if s == nil {
		return
	}
	s.setInt(settingMarketOrdersRetentionDays, v)
}

func (s *Settings) SysTrayEnabled() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingSysTrayEnabled, settingSysTrayEnabledDefault)
}

func (s *Settings) SysTrayEnabledDefault() bool {
	return settingSysTrayEnabledDefault
}

func (s *Settings) SetSysTrayEnabled(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingSysTrayEnabled, v)
}

func (s *Settings) WindowSize() fyne.Size {
	if s == nil {
		return fyne.Size{}
	}
	x := s.getFloatList(settingWindowsSize, nil)
	if len(x) < 2 {
		return fyne.NewSize(settingWindowWidthDefault, settingWindowHeightDefault)
	}
	return fyne.NewSize(float32(x[0]), float32(x[1]))
}

func (s *Settings) ResetWindowSize() {
	if s == nil {
		return
	}
	s.SetWindowSize(fyne.NewSize(settingWindowWidthDefault, settingWindowHeightDefault))
}

func (s *Settings) SetWindowSize(v fyne.Size) {
	if s == nil {
		return
	}
	s.setFloatList(settingWindowsSize, []float64{float64(v.Width), float64(v.Height)})
}

func (s *Settings) ResetTabsMainID() {
	if s == nil {
		return
	}
	s.SetTabsMainID(settingTabsMainIDDefault)
}

func (s *Settings) SetTabsMainID(v int) {
	if s == nil {
		return
	}
	s.setInt(settingTabsMainID, v)
}

func (s *Settings) LastCharacterID() int64 {
	if s == nil {
		return 0
	}
	return s.getInt64(settingLastCharacterID, 0)
}

func (s *Settings) ResetLastCharacterID() {
	if s == nil {
		return
	}
	s.SetLastCharacterID(0)
}

func (s *Settings) SetLastCharacterID(id int64) {
	if s == nil {
		return
	}
	s.setInt64(settingLastCharacterID, id)
}

func (s *Settings) LastCorporationID() int64 {
	if s == nil {
		return 0
	}
	return s.getInt64(settingLastCorporationID, 0)
}

func (s *Settings) ResetLastCorporationID() {
	if s == nil {
		return
	}
	s.SetLastCorporationID(0)
}

func (s *Settings) SetLastCorporationID(id int64) {
	if s == nil {
		return
	}
	s.setInt64(settingLastCorporationID, id)
}

func (s *Settings) MaxWalletTransactions() int {
	if s == nil {
		return 0
	}
	return s.getInt(settingMaxWalletTransactions, settingMaxWalletTransactionsDefault)
}

func (s *Settings) MaxWalletTransactionsPresets() (minimum int, maximum int, def int) {
	minimum = 0
	maximum = settingMaxWalletTransactionsMax
	def = settingMaxWalletTransactionsDefault
	return
}

func (s *Settings) ResetMaxWalletTransactions() {
	if s == nil {
		return
	}
	s.SetMaxWalletTransactions(settingMaxWalletTransactionsDefault)
}

func (s *Settings) SetMaxWalletTransactions(v int) {
	if s == nil {
		return
	}
	s.setInt(settingMaxWalletTransactions, v)
}

func (s *Settings) NotifyTimeoutHours() int {
	if s == nil {
		return 0
	}
	return s.getInt(settingNotifyTimeoutHours, settingNotifyTimeoutHoursDefault)
}

func (s *Settings) NotifyTimeoutHoursPresets() (minimum int, maximum int, def int) {
	minimum = settingNotifyTimeoutHoursMin
	maximum = settingNotifyTimeoutHoursMax
	def = settingNotifyTimeoutHoursDefault
	return
}

func (s *Settings) ResetNotifyTimeoutHours() {
	if s == nil {
		return
	}
	s.SetNotifyTimeoutHours(settingNotifyTimeoutHoursDefault)
}

func (s *Settings) SetNotifyTimeoutHours(v int) {
	if s == nil {
		return
	}
	s.setInt(settingNotifyTimeoutHours, v)
}

func (s *Settings) NotificationTypesEnabled() set.Set[string] {
	if s == nil {
		return set.Set[string]{}
	}
	return set.Of(s.getStringList(settingNotificationTypesEnabled, nil)...)
}

func (s *Settings) ResetNotificationTypesEnabled() {
	if s == nil {
		return
	}
	s.SetNotificationTypesEnabled(set.Of[string]())
}

func (s *Settings) SetNotificationTypesEnabled(v set.Set[string]) {
	if s == nil {
		return
	}
	s.setStringList(settingNotificationTypesEnabled, slices.Collect(v.All()))
}

func (s *Settings) NotifyCommunicationsEarliest() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.calcNotifyEarliest(settingNotifyCommunicationsEarliest)
}

func (s *Settings) SetNotifyCommunicationsEarliest(t time.Time) {
	if s == nil {
		return
	}
	s.setTime(settingNotifyCommunicationsEarliest, t)
}

func (s *Settings) NotifyContractsEarliest() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.calcNotifyEarliest(settingNotifyContractsEarliest)
}

func (s *Settings) SetNotifyContractsEarliest(t time.Time) {
	if s == nil {
		return
	}
	s.setTime(settingNotifyContractsEarliest, t)
}

func (s *Settings) NotifyMailsEarliest() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.calcNotifyEarliest(settingNotifyMailsEarliest)
}

func (s *Settings) SetNotifyMailsEarliest(t time.Time) {
	if s == nil {
		return
	}
	s.setTime(settingNotifyMailsEarliest, t)
}

func (s *Settings) NotifyPIEarliest() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.calcNotifyEarliest(settingNotifyPIEarliest)
}

func (s *Settings) SetNotifyPIEarliest(t time.Time) {
	if s == nil {
		return
	}
	s.setTime(settingNotifyPIEarliest, t)
}

func (s *Settings) NotifyTrainingEarliest() time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.calcNotifyEarliest(settingNotifyTrainingEarliest)
}

func (s *Settings) SetNotifyTrainingEarliest(t time.Time) {
	if s == nil {
		return
	}
	s.setTime(settingNotifyTrainingEarliest, t)
}

// calcNotifyEarliest returns the earliest time for a class of notifications.
// Might return a zero time in some circumstances.
func (s *Settings) calcNotifyEarliest(key string) time.Time {
	earliest := s.getTime(key, time.Time{})
	if earliest.IsZero() {
		// Recording the earliest when enabling a switch was added later for mails and communications
		// This workaround avoids a potential notification spam from older items.
		earliest = time.Now().UTC().Add(-notifyEarliestFallback)
		s.setTime(key, earliest)
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

func (s *Settings) NotifyCommunicationsEnabled() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingNotifyCommunicationsEnabled, settingNotifyCommunicationsEnabledDefault)
}

func (s *Settings) NotifyCommunicationsEnabledDefault() bool {
	if s == nil {
		return false
	}
	return settingNotifyCommunicationsEnabledDefault
}

func (s *Settings) SetNotifyCommunicationsEnabled(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingNotifyCommunicationsEnabled, v)
}

func (s *Settings) NotifyContractsEnabled() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingNotifyContractsEnabled, settingNotifyContractsEnabledDefault)
}

func (s *Settings) NotifyContractsEnabledDefault() bool {
	if s == nil {
		return false
	}
	return settingNotifyContractsEnabledDefault
}

func (s *Settings) SetNotifyContractsEnabled(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingNotifyContractsEnabled, v)
}

func (s *Settings) NotifyMailsEnabled() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingNotifyMailsEnabled, settingNotifyMailsEnabledDefault)
}

func (s *Settings) NotifyMailsEnabledDefault() bool {
	if s == nil {
		return false
	}
	return settingNotifyMailsEnabledDefault
}

func (s *Settings) SetNotifyMailsEnabled(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingNotifyMailsEnabled, v)
}

func (s *Settings) NotifyPIEnabled() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingNotifyPIEnabled, settingNotifyPIEnabledDefault)
}

func (s *Settings) NotifyPIEnabledDefault() bool {
	if s == nil {
		return false
	}
	return settingNotifyPIEnabledDefault
}

func (s *Settings) SetNotifyPIEnabled(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingNotifyPIEnabled, v)
}

func (s *Settings) NotifyTrainingEnabled() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingNotifyTrainingEnabled, settingNotifyTrainingEnabledDefault)
}

func (s *Settings) NotifyTrainingEnabledDefault() bool {
	if s == nil {
		return false
	}
	return settingNotifyTrainingEnabledDefault
}

func (s *Settings) SetNotifyTrainingEnabled(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingNotifyTrainingEnabled, v)
}

func (s *Settings) TabsMainID() int {
	if s == nil {
		return 0
	}
	return s.getInt(settingTabsMainID, settingTabsMainIDDefault)
}

func (s *Settings) RecentSearches() []int64 {
	if s == nil {
		return nil
	}
	// Stored as strings rather than via a native int-list API: the platform's int type
	// is 32-bit on some platforms (e.g. Android) and would truncate EVE IDs above
	// math.MaxInt32.
	raw := s.getStringList(settingRecentSearches, nil)
	out := make([]int64, 0, len(raw))
	for _, x := range raw {
		v, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}

func (s *Settings) SetRecentSearches(v []int64) {
	if s == nil {
		return
	}
	// Stored as strings; see RecentSearches for why.
	s.setStringList(settingRecentSearches, xslices.Map(v, func(x int64) string {
		return strconv.FormatInt(x, 10)
	}))
}

func (s *Settings) PreferMarketTab() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingPreferMarketTab, false)
}

func (s *Settings) ResetPreferMarketTab() {
	if s == nil {
		return
	}
	s.SetPreferMarketTab(false)
}

func (s *Settings) SetPreferMarketTab(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingPreferMarketTab, v)
}

func (s *Settings) HideLimitedCorporations() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingHideLimitedCorporations, false)
}

func (s *Settings) HideLimitedCorporationsDefault() bool {
	if s == nil {
		return false
	}
	return settingHideLimitedCorporationsDefault
}

func (s *Settings) SetHideLimitedCorporations(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingHideLimitedCorporations, v)
}

func (s *Settings) ColorTheme() ColorTheme {
	if s == nil {
		return ColorTheme("")
	}
	x := s.getString(settingColorTheme, string(settingColorThemeDefault))
	return ColorTheme(x)
}

func (s *Settings) ColorThemeDefault() ColorTheme {
	if s == nil {
		return ColorTheme("")
	}
	return settingColorThemeDefault
}

func (s *Settings) ResetColorTheme() {
	if s == nil {
		return
	}
	s.SetColorTheme(settingColorThemeDefault)
}

func (s *Settings) SetColorTheme(v ColorTheme) {
	if s == nil {
		return
	}
	s.setString(settingColorTheme, string(v))
}

func (s *Settings) FyneScale() float64 {
	if s == nil {
		return 0
	}
	return s.getFloat(settingFyneScale, settingFyneScaleDefault)
}

func (s *Settings) FyneScaleDefault() float64 {
	if s == nil {
		return 0
	}
	return settingFyneScaleDefault
}

func (s *Settings) ResetFyneScale() {
	if s == nil {
		return
	}
	s.SetFyneScale(settingFyneScaleDefault)
}

func (s *Settings) SetFyneScale(v float64) {
	if s == nil {
		return
	}
	s.setFloat(settingFyneScale, v)
}

func (s *Settings) DisableDPIDetection() bool {
	if s == nil {
		return false
	}
	return s.getBool(settingDisableDPIDetection, false)
}

func (s *Settings) ResetDisableDPIDetection() {
	if s == nil {
		return
	}
	s.SetDisableDPIDetection(false)
}

func (s *Settings) SetDisableDPIDetection(v bool) {
	if s == nil {
		return
	}
	s.setBool(settingDisableDPIDetection, v)
}

// ResetUI resets all UI related settings to default.
func (s *Settings) ResetUI() {
	if s == nil {
		return
	}
	s.ResetTabsMainID()
	s.ResetWindowSize()
	s.ResetColorTheme()
	s.ResetFyneScale()
	s.ResetDisableDPIDetection()
}
