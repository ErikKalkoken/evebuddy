package core

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
)

func TestMakeManageMenu(t *testing.T) {
	// On disk, because settings are persisted in a goroutine.
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()

	// itemLabels returns the labels of all items in m. Separators are shown as "-".
	itemLabels := func(m *fyne.Menu) []string {
		var s []string
		for _, it := range m.Items {
			if it.IsSeparator {
				s = append(s, "-")
			} else {
				s = append(s, it.Label)
			}
		}
		return s
	}

	// findItem returns the item with the given label in m.
	findItem := func(t *testing.T, m *fyne.Menu, label string) *fyne.MenuItem {
		t.Helper()
		for _, it := range m.Items {
			if it.Label == label {
				return it
			}
		}
		t.Fatalf("menu item not found: %s", label)
		return nil
	}

	t.Run("should show items without close when systray is disabled", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.settings.SetSysTrayEnabled(false)
		// when
		m := makeManageMenu(du)
		// then
		want := []string{"Settings", "Manage Characters", "Update Status", "-", "User Data", "About", "-", "Quit"}
		xassert.Equal(t, want, itemLabels(m))
	})
	t.Run("should show items with close when systray is enabled", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.settings.SetSysTrayEnabled(true)
		// when
		m := makeManageMenu(du)
		// then
		want := []string{"Settings", "Manage Characters", "Update Status", "-", "User Data", "About", "-", "Close", "Quit"}
		xassert.Equal(t, want, itemLabels(m))
	})
	t.Run("should define shortcut for close", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.settings.SetSysTrayEnabled(true)
		// when
		m := makeManageMenu(du)
		// then
		want := &desktop.CustomShortcut{KeyName: fyne.KeyF4, Modifier: fyne.KeyModifierAlt}
		xassert.Equal[fyne.Shortcut](t, want, findItem(t, m, "Close").Shortcut)
	})
	t.Run("should run shortcut handler when item is tapped", func(t *testing.T) {
		// given
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		var called []string
		for _, name := range []string{"settings", "manageCharacters", "updateStatus", "quit"} {
			xdesktop.AddShortcut(name, xdesktop.ShortcutWithHandler{
				Shortcut: &desktop.CustomShortcut{KeyName: fyne.KeyA, Modifier: fyne.KeyModifierAlt},
				Handler: func(_ fyne.Shortcut) {
					called = append(called, name)
				},
			}, du.MainWindow())
		}
		m := makeManageMenu(du)
		// when
		for _, label := range []string{"Settings", "Manage Characters", "Update Status", "Quit"} {
			it := findItem(t, m, label)
			require.NotNil(t, it.Shortcut, label)
			it.Action()
		}
		// then
		xassert.Equal(t, []string{"settings", "manageCharacters", "updateStatus", "quit"}, called)
	})
}
