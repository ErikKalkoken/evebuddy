package evenotification_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/stretchr/testify/assert"
)

func TestType(t *testing.T) {
	t.Run("can convert to string", func(t *testing.T) {
		x := evenotification.BountyClaimMsg
		assert.Equal(t, "BountyClaimMsg", x.String())
	})
	t.Run("can convert to display string", func(t *testing.T) {
		x := evenotification.SovereigntyTCUDamageMsg
		assert.Equal(t, "Sovereignty TCU Damage Msg", x.Display())
	})
}
