package evenotification

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// fromLDAPTime converts an ldap time to golang time
func fromLDAPTime(ldapTime int64) time.Time {
	return time.Unix((ldapTime/10000000)-11644473600, 0).UTC()
}

// fromLDAPDuration converts an ldap duration to golang duration
func fromLDAPDuration(ldapDuration int64) time.Duration {
	return time.Duration(ldapDuration/10) * time.Microsecond
}

// makeInfoLink2 returns an info link from link data.
// It returns just the name if a link can not be constructed.
func makeInfoLink2(name string, linkData []any) string {
	if len(linkData) != 2 && len(linkData) != 3 {
		return name
	}
	if linkData[0] != "showinfo" {
		return name
	}
	var link string
	switch len(linkData) {
	case 2:
		link = fmt.Sprintf("showinfo:%d", linkData[1])
	case 3:
		link = fmt.Sprintf("showinfo:%d//%d", linkData[1], linkData[2])

	}
	return makeMarkDownLink(name, link)
}

type eveEntityConvertible interface {
	ToEveEntity() *app.EveEntity
}

// makeInfoLink returns an info link for an entity.
// It returns the name if a link can not be constructed.
func makeInfoLink(x eveEntityConvertible) string {
	o := x.ToEveEntity()
	if o == nil {
		return "?"
	}
	link, err := o.InfoLink()
	if err != nil {
		slog.Warn("Failed to resolve link for entity in notification", "entityID", o.ID, "error", err)
		if o.Name == "" {
			return "?"
		}
		return o.Name
	}
	return makeMarkDownLink(o.Name, link)
}

// makeMarkDownLink returns a link in markdown syntax.
// It returns just the label when url is empty.
func makeMarkDownLink(label, url string) string {
	if url == "" {
		return label
	}
	return fmt.Sprintf("[%s](%s)", label, url)
}
