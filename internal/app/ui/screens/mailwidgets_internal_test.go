package screens

import "github.com/ErikKalkoken/evebuddy/internal/xwidget"

// headerShowsOwner reports whether a mail header shows the owner with the given name.
func headerShowsOwner(w *MailHeaderWidget, name string) bool {
	for _, o := range w.recipients.Objects {
		if x, ok := o.(*xwidget.TappableLabel); ok && x.Text == "["+name+"]" {
			return true
		}
	}
	return false
}
