// Package xdesktop extends Fyne's desktop package.
package xdesktop

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
)

// ShortcutWithHandler represents a Fyne shortcut with it's handler.
type ShortcutWithHandler struct {
	Shortcut fyne.Shortcut
	Handler  func(shortcut fyne.Shortcut)
}

var shortcuts = map[fyne.Window]map[string]ShortcutWithHandler{}

// AddShortcut adds a shortcut sc to a window w.
func AddShortcut(name string, sc ShortcutWithHandler, w fyne.Window) {
	if name == "" {
		fyne.LogError("adding shortcut", fmt.Errorf("name missing"))
		return
	}
	if sc.Shortcut == nil {
		fyne.LogError("adding shortcut", fmt.Errorf("shortcut missing"))
		return
	}
	if sc.Handler == nil {
		fyne.LogError("adding shortcut", fmt.Errorf("handler missing"))
		return
	}
	if w == nil {
		fyne.LogError("adding shortcut", fmt.Errorf("window missing"))
		return
	}
	if shortcuts[w] == nil {
		shortcuts[w] = make(map[string]ShortcutWithHandler)
	}
	shortcuts[w][name] = sc
}

// DisableShortcuts disables all shortcuts in a window w.
func DisableShortcuts(w fyne.Window) {
	if w == nil {
		fyne.LogError("disable shortcuts", fmt.Errorf("window missing"))
		return
	}
	for _, sc := range shortcuts[w] {
		w.Canvas().RemoveShortcut(sc.Shortcut)
	}
}

// DisableShortcutsForDialog disables all shortcuts temporarily
// for a dialog d in window w.
// Shortcuts will be re-enabled once the dialog is closed.
func DisableShortcutsForDialog(d dialog.Dialog, w fyne.Window) {
	if d == nil {
		fyne.LogError("disable shortcuts for dialog", fmt.Errorf("dialog missing"))
		return
	}
	if w == nil {
		fyne.LogError("disable shortcuts for dialog", fmt.Errorf("window missing"))
		return
	}
	if len(shortcuts[w]) == 0 {
		return
	}
	kxdialog.AddDialogKeyHandler(d, w)
	DisableShortcuts(w)
	d.SetOnClosed(func() {
		EnableShortcuts(w)
	})
}

// EnableShortcuts enables all shortcuts in a window w.
func EnableShortcuts(w fyne.Window) {
	if w == nil {
		fyne.LogError("enable shortcuts", fmt.Errorf("window missing"))
		return
	}
	for _, sc := range shortcuts[w] {
		w.Canvas().AddShortcut(sc.Shortcut, sc.Handler)
	}
}

// RemoveAllShortcuts removes all shortcuts from a window w.
func RemoveAllShortcuts(w fyne.Window) {
	if w == nil {
		fyne.LogError("remove all shortcuts", fmt.Errorf("window missing"))
		return
	}
	for name := range shortcuts[w] {
		RemoveShortcut(name, w)
	}
}

// RemoveShortcut tries to remove a shortcut from a window w and returns it when found.
func RemoveShortcut(name string, w fyne.Window) ShortcutWithHandler {
	if name == "" {
		fyne.LogError("remove shortcut", fmt.Errorf("name missing"))
		return ShortcutWithHandler{}
	}
	if w == nil {
		fyne.LogError("remove shortcut", fmt.Errorf("window missing"))
		return ShortcutWithHandler{}
	}
	m, ok := shortcuts[w]
	if !ok {
		return ShortcutWithHandler{}
	}
	def, ok := m[name]
	if !ok {
		return ShortcutWithHandler{}
	}
	delete(m, name)
	return def
}

// Shortcut tries to return a shortcut and reports whether it was found.
func Shortcut(name string, w fyne.Window) (ShortcutWithHandler, bool) {
	if name == "" {
		fyne.LogError("find shortcut", fmt.Errorf("name missing"))
		return ShortcutWithHandler{}, false
	}
	if w == nil {
		fyne.LogError("find shortcut", fmt.Errorf("window missing"))
		return ShortcutWithHandler{}, false
	}
	m, ok := shortcuts[w]
	if !ok {
		return ShortcutWithHandler{}, false
	}
	def, ok := m[name]
	if !ok {
		return ShortcutWithHandler{}, false
	}
	return def, true
}
