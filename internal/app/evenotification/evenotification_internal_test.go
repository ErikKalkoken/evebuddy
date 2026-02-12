package evenotification

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestLDAPTimeConversion(t *testing.T) {
	t.Run("should convert LDAP time", func(t *testing.T) {
		x := fromLDAPTime(131924601300000000)
	xassert.Equal(t, time.Date(2019, 1, 20, 12, 15, 30, 0, time.UTC), x)
	})
	t.Run("should convert LDAP duration", func(t *testing.T) {
		x := fromLDAPDuration(9000000000)
	xassert.Equal(t, time.Duration(15*time.Minute), x)
	})
}
