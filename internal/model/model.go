// Package model contains the entity objects, which are used across the app.
package model

// Setting keys
const (
	SettingLastCharacterID       = "settings-lastCharacterID"
	SettingMaxMails              = "settings-maxMails"
	SettingMaxWalletTransactions = "settings-maxWalletTransactions"
	SettingTheme                 = "settings-theme"
	ThemeAuto                    = "Auto"
	ThemeDark                    = "Dark"
	ThemeLight                   = "Light"
)

// Default settings
const (
	SettingMaxMailsDefault              = 1_000
	SettingMaxWalletTransactionsDefault = 10_000
)

// EntityShort is a short representation of an entity.
type EntityShort[T int | int32 | int64] struct {
	ID   T
	Name string
}

// Position is a position in 3D space.
type Position struct {
	X float64
	Y float64
	Z float64
}
