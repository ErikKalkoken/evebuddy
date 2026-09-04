package evenotification_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestContact_RenderESI(t *testing.T) {
	en := evenotification.New(nil)

	t.Run("ContactAdd", func(t *testing.T) {
		text := "level: 3\nmessage: 'Welcome aboard!'\n"
		title, body, err := en.RenderESI(t.Context(), app.ContactAdd, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Contact added", title)
		assert.Contains(t, body, "3")
		assert.Contains(t, body, "Welcome aboard!")
	})

	t.Run("ContactEdit", func(t *testing.T) {
		text := "level: 10.0\nmessage: 'Standing updated'\n"
		title, body, err := en.RenderESI(t.Context(), app.ContactEdit, optional.New(text), time.Now())
		require.NoError(t, err)
		assert.Equal(t, "Contact standing updated", title)
		assert.Contains(t, body, "10")
		assert.Contains(t, body, "Standing updated")
	})
}
