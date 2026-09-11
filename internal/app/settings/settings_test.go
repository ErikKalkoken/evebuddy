package settings_test

import (
	"log/slog"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSettings(t *testing.T) {
	t.Run("Window size", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := fyne.NewSize(123, 456)
		s.SetWindowSize(x)
	xassert.Equal(t, x, s.WindowSize())
	})
	t.Run("Log level", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := "debug"
		s.SetLogLevel(x)
	xassert.Equal(t, x, s.LogLevel())
	xassert.Equal(t, slog.LevelDebug, s.LogLevelSlog())
	})
	t.Run("RecentSearches", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := []int64{1, 2, 3, 2_200_000_000} // last one beyond int32 range, e.g. a valid EVE ID
		s.SetRecentSearches(x)
	xassert.Equal(t, x, s.RecentSearches())
	})
	t.Run("NotificationTypesEnabled", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		got := set.Of([]string{"alpha", "bravo"}...)
		s.SetNotificationTypesEnabled(got)
		want := s.NotificationTypesEnabled()
		xassert.Equal(t, want, got)
	})
	t.Run("LastCharacterID", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := int64(2_200_000_000) // beyond int32 range, e.g. a valid EVE character ID
		s.SetLastCharacterID(x)
	xassert.Equal(t, x, s.LastCharacterID())
	})
	t.Run("LastCorporationID", func(t *testing.T) {
		p := settings.NewMyPref()
		s := settings.New(p)
		x := int64(2_200_000_000) // beyond int32 range, e.g. a valid EVE corporation ID
		s.SetLastCorporationID(x)
	xassert.Equal(t, x, s.LastCorporationID())
	})
}

func TestColorTheme(t *testing.T) {
	t.Run("Default theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		x1 := s.ColorTheme()
	xassert.Equal(t, settings.Auto, x1)
	})
	t.Run("Can set and get theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		s.SetColorTheme(settings.Dark)
		x1 := s.ColorTheme()
	xassert.Equal(t, settings.Dark, x1)
	})
	t.Run("Can reset theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		s.SetColorTheme(settings.Dark)
		s.ResetColorTheme()
		assert.Equal(t, settings.Auto, s.ColorTheme())
	})
	t.Run("ColorThemeDefault reports the default theme", func(t *testing.T) {
		s := settings.New(settings.NewMyPref())
		assert.Equal(t, settings.Auto, s.ColorThemeDefault())
	})
}

func TestDeveloperMode(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.False(t, s.DeveloperMode())
	s.SetDeveloperMode(true)
	assert.True(t, s.DeveloperMode())
}

func TestLogLevelNames(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	want := []string{"debug", "info", "warning", "error"}
	assert.Equal(t, want, s.LogLevelNames())
}

func TestLogLevelDefaultAndReset(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.Equal(t, "info", s.LogLevelDefault())
	assert.Equal(t, s.LogLevelDefault(), s.LogLevel())
	s.SetLogLevel("debug")
	assert.Equal(t, "debug", s.LogLevel())
	s.ResetLogLevel()
	assert.Equal(t, s.LogLevelDefault(), s.LogLevel())
}

func TestApprovedContactCost(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	minimum, maximum, def := s.ApprovedContactCostPresets()
	assert.Equal(t, 0, minimum)
	assert.Equal(t, 1_000_000, maximum)
	assert.Equal(t, 0, def)
	assert.Equal(t, def, s.ApprovedContactCost())
	s.SetApprovedContactCost(500)
	assert.Equal(t, 500, s.ApprovedContactCost())
}

func TestMaxMails(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	minimum, maximum, def := s.MaxMailsPresets()
	assert.Equal(t, 0, minimum)
	assert.Equal(t, 10_000, maximum)
	assert.Equal(t, 250, def)
	assert.Equal(t, def, s.MaxMails())
	s.SetMaxMails(500)
	assert.Equal(t, 500, s.MaxMails())
}

func TestMarketOrderRetentionDays(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	minimum, maximum, def := s.MarketOrderRetentionDaysPresets()
	assert.Equal(t, 30, minimum)
	assert.Equal(t, 360, maximum)
	assert.Equal(t, 90, def)
	assert.Equal(t, def, s.MarketOrderRetentionDays())
	s.SetMarketOrdersRetentionDay(120)
	assert.Equal(t, 120, s.MarketOrderRetentionDays())
}

func TestSysTrayEnabled(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.True(t, s.SysTrayEnabledDefault())
	assert.Equal(t, s.SysTrayEnabledDefault(), s.SysTrayEnabled())
	s.SetSysTrayEnabled(false)
	assert.False(t, s.SysTrayEnabled())
}

func TestWindowSizeDefaultAndReset(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	want := fyne.NewSize(1000, 600)
	xassert.Equal(t, want, s.WindowSize())
	s.SetWindowSize(fyne.NewSize(200, 300))
	xassert.Equal(t, fyne.NewSize(200, 300), s.WindowSize())
	s.ResetWindowSize()
	xassert.Equal(t, want, s.WindowSize())
}

func TestTabsMainID(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.Equal(t, -1, s.TabsMainID())
	s.SetTabsMainID(3)
	assert.Equal(t, 3, s.TabsMainID())
	s.ResetTabsMainID()
	assert.Equal(t, -1, s.TabsMainID())
}

func TestLastCharacterIDReset(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	s.SetLastCharacterID(123)
	assert.Equal(t, int64(123), s.LastCharacterID())
	s.ResetLastCharacterID()
	assert.Equal(t, int64(0), s.LastCharacterID())
}

func TestLastCorporationIDReset(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	s.SetLastCorporationID(123)
	assert.Equal(t, int64(123), s.LastCorporationID())
	s.ResetLastCorporationID()
	assert.Equal(t, int64(0), s.LastCorporationID())
}

func TestMaxWalletTransactions(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	minimum, maximum, def := s.MaxWalletTransactionsPresets()
	assert.Equal(t, 0, minimum)
	assert.Equal(t, 10_000, maximum)
	assert.Equal(t, 1_000, def)
	assert.Equal(t, def, s.MaxWalletTransactions())
	s.SetMaxWalletTransactions(500)
	assert.Equal(t, 500, s.MaxWalletTransactions())
	s.ResetMaxWalletTransactions()
	assert.Equal(t, def, s.MaxWalletTransactions())
}

func TestNotifyTimeoutHours(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	minimum, maximum, def := s.NotifyTimeoutHoursPresets()
	assert.Equal(t, 1, minimum)
	assert.Equal(t, 90*24, maximum)
	assert.Equal(t, 30*24, def)
	assert.Equal(t, def, s.NotifyTimeoutHours())
	s.SetNotifyTimeoutHours(48)
	assert.Equal(t, 48, s.NotifyTimeoutHours())
	s.ResetNotifyTimeoutHours()
	assert.Equal(t, def, s.NotifyTimeoutHours())
}

func TestNotificationTypesEnabledReset(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	s.SetNotificationTypesEnabled(set.Of("alpha", "bravo"))
	assert.Equal(t, 2, s.NotificationTypesEnabled().Size())
	s.ResetNotificationTypesEnabled()
	assert.Equal(t, 0, s.NotificationTypesEnabled().Size())
}

func TestNotifyEarliestGettersAndSetters(t *testing.T) {
	cases := []struct {
		name string
		get  func(*settings.Settings) time.Time
		set  func(*settings.Settings, time.Time)
	}{
		{"Communications", (*settings.Settings).NotifyCommunicationsEarliest, (*settings.Settings).SetNotifyCommunicationsEarliest},
		{"Contracts", (*settings.Settings).NotifyContractsEarliest, (*settings.Settings).SetNotifyContractsEarliest},
		{"Mails", (*settings.Settings).NotifyMailsEarliest, (*settings.Settings).SetNotifyMailsEarliest},
		{"PI", (*settings.Settings).NotifyPIEarliest, (*settings.Settings).SetNotifyPIEarliest},
		{"Training", (*settings.Settings).NotifyTrainingEarliest, (*settings.Settings).SetNotifyTrainingEarliest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := settings.New(settings.NewMyPref())
			want := time.Now().UTC().Add(-time.Hour)
			tc.set(s, want)
			got := tc.get(s)
			assert.WithinDuration(t, want, got, time.Second)
		})
	}
}

func TestNotifyEnabledFlags(t *testing.T) {
	cases := []struct {
		name       string
		get        func(*settings.Settings) bool
		getDefault func(*settings.Settings) bool
		set        func(*settings.Settings, bool)
	}{
		{"Communications", (*settings.Settings).NotifyCommunicationsEnabled, (*settings.Settings).NotifyCommunicationsEnabledDefault, (*settings.Settings).SetNotifyCommunicationsEnabled},
		{"Contracts", (*settings.Settings).NotifyContractsEnabled, (*settings.Settings).NotifyContractsEnabledDefault, (*settings.Settings).SetNotifyContractsEnabled},
		{"Mails", (*settings.Settings).NotifyMailsEnabled, (*settings.Settings).NotifyMailsEnabledDefault, (*settings.Settings).SetNotifyMailsEnabled},
		{"PI", (*settings.Settings).NotifyPIEnabled, (*settings.Settings).NotifyPIEnabledDefault, (*settings.Settings).SetNotifyPIEnabled},
		{"Training", (*settings.Settings).NotifyTrainingEnabled, (*settings.Settings).NotifyTrainingEnabledDefault, (*settings.Settings).SetNotifyTrainingEnabled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := settings.New(settings.NewMyPref())
			assert.False(t, tc.getDefault(s))
			assert.Equal(t, tc.getDefault(s), tc.get(s))
			tc.set(s, true)
			assert.True(t, tc.get(s))
		})
	}
}

func TestPreferMarketTab(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.False(t, s.PreferMarketTab())
	s.SetPreferMarketTab(true)
	assert.True(t, s.PreferMarketTab())
	s.ResetPreferMarketTab()
	assert.False(t, s.PreferMarketTab())
}

func TestHideLimitedCorporations(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.False(t, s.HideLimitedCorporationsDefault())
	assert.Equal(t, s.HideLimitedCorporationsDefault(), s.HideLimitedCorporations())
	s.SetHideLimitedCorporations(true)
	assert.True(t, s.HideLimitedCorporations())
}

func TestFyneScale(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.Equal(t, 1.0, s.FyneScaleDefault())
	assert.Equal(t, s.FyneScaleDefault(), s.FyneScale())
	s.SetFyneScale(1.5)
	assert.Equal(t, 1.5, s.FyneScale())
	s.ResetFyneScale()
	assert.Equal(t, 1.0, s.FyneScale())
}

func TestDisableDPIDetection(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	assert.False(t, s.DisableDPIDetection())
	s.SetDisableDPIDetection(true)
	assert.True(t, s.DisableDPIDetection())
	s.ResetDisableDPIDetection()
	assert.False(t, s.DisableDPIDetection())
}

func TestResetUI(t *testing.T) {
	s := settings.New(settings.NewMyPref())
	s.SetTabsMainID(5)
	s.SetWindowSize(fyne.NewSize(200, 300))
	s.SetColorTheme(settings.Dark)
	s.SetFyneScale(2.0)
	s.SetDisableDPIDetection(true)

	s.ResetUI()

	assert.Equal(t, -1, s.TabsMainID())
	xassert.Equal(t, fyne.NewSize(1000, 600), s.WindowSize())
	assert.Equal(t, settings.Auto, s.ColorTheme())
	assert.Equal(t, 1.0, s.FyneScale())
	assert.False(t, s.DisableDPIDetection())
}

func TestKeys(t *testing.T) {
	got := settings.Keys()
	assert.NotEmpty(t, got)
	seen := set.Of(got...)
	assert.Equal(t, len(got), seen.Size(), "Keys() should not contain duplicate entries")
}

func TestSettingsNilReceiverIsSafe(t *testing.T) {
	var s *settings.Settings

	assert.NotPanics(t, func() {
		assert.False(t, s.DeveloperMode())
		s.SetDeveloperMode(true)

		assert.Nil(t, s.LogLevelNames())
		assert.Equal(t, slog.Level(0), s.LogLevelSlog())
		assert.Equal(t, "", s.LogLevel())
		assert.Equal(t, "", s.LogLevelDefault())
		s.ResetLogLevel()
		s.SetLogLevel("debug")

		assert.Equal(t, 0, s.ApprovedContactCost())
		s.SetApprovedContactCost(1)

		assert.Equal(t, 0, s.MaxMails())
		s.SetMaxMails(1)

		assert.Equal(t, 0, s.MarketOrderRetentionDays())
		s.SetMarketOrdersRetentionDay(1)

		assert.False(t, s.SysTrayEnabled())
		assert.True(t, s.SysTrayEnabledDefault())
		s.SetSysTrayEnabled(true)

		assert.Equal(t, fyne.Size{}, s.WindowSize())
		s.ResetWindowSize()
		s.SetWindowSize(fyne.NewSize(1, 1))

		s.ResetTabsMainID()
		s.SetTabsMainID(1)
		assert.Equal(t, 0, s.TabsMainID())

		assert.Equal(t, int64(0), s.LastCharacterID())
		s.ResetLastCharacterID()
		s.SetLastCharacterID(1)

		assert.Equal(t, int64(0), s.LastCorporationID())
		s.ResetLastCorporationID()
		s.SetLastCorporationID(1)

		assert.Equal(t, 0, s.MaxWalletTransactions())
		s.ResetMaxWalletTransactions()
		s.SetMaxWalletTransactions(1)

		assert.Equal(t, 0, s.NotifyTimeoutHours())
		s.ResetNotifyTimeoutHours()
		s.SetNotifyTimeoutHours(1)

		assert.Equal(t, 0, s.NotificationTypesEnabled().Size())
		s.ResetNotificationTypesEnabled()
		s.SetNotificationTypesEnabled(set.Of("x"))

		for _, get := range []func() time.Time{
			s.NotifyCommunicationsEarliest,
			s.NotifyContractsEarliest,
			s.NotifyMailsEarliest,
			s.NotifyPIEarliest,
			s.NotifyTrainingEarliest,
		} {
			assert.True(t, get().IsZero())
		}
		s.SetNotifyCommunicationsEarliest(time.Now())
		s.SetNotifyContractsEarliest(time.Now())
		s.SetNotifyMailsEarliest(time.Now())
		s.SetNotifyPIEarliest(time.Now())
		s.SetNotifyTrainingEarliest(time.Now())

		for _, get := range []func() bool{
			s.NotifyCommunicationsEnabled, s.NotifyCommunicationsEnabledDefault,
			s.NotifyContractsEnabled, s.NotifyContractsEnabledDefault,
			s.NotifyMailsEnabled, s.NotifyMailsEnabledDefault,
			s.NotifyPIEnabled, s.NotifyPIEnabledDefault,
			s.NotifyTrainingEnabled, s.NotifyTrainingEnabledDefault,
		} {
			assert.False(t, get())
		}
		s.SetNotifyCommunicationsEnabled(true)
		s.SetNotifyContractsEnabled(true)
		s.SetNotifyMailsEnabled(true)
		s.SetNotifyPIEnabled(true)
		s.SetNotifyTrainingEnabled(true)

		assert.Nil(t, s.RecentSearches())
		s.SetRecentSearches([]int64{1})

		assert.False(t, s.PreferMarketTab())
		s.ResetPreferMarketTab()
		s.SetPreferMarketTab(true)

		assert.False(t, s.HideLimitedCorporations())
		assert.False(t, s.HideLimitedCorporationsDefault())
		s.SetHideLimitedCorporations(true)

		assert.Equal(t, settings.ColorTheme(""), s.ColorTheme())
		assert.Equal(t, settings.ColorTheme(""), s.ColorThemeDefault())
		s.ResetColorTheme()
		s.SetColorTheme(settings.Dark)

		assert.Equal(t, float64(0), s.FyneScale())
		assert.Equal(t, float64(0), s.FyneScaleDefault())
		s.ResetFyneScale()
		s.SetFyneScale(1)

		assert.False(t, s.DisableDPIDetection())
		s.ResetDisableDPIDetection()
		s.SetDisableDPIDetection(true)

		s.ResetUI()
	})
}
