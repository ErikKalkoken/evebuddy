package storage

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestNotificationHelpers(t *testing.T) {
	st := &Storage{}
	t.Run("can convert valid name to type", func(t *testing.T) {
		got, found := EveNotificationTypeFromESIString("StructureDestroyed")
		if assert.True(t, found) {
			assert.Equal(t, app.StructureDestroyed, got)
		}
	})
	t.Run("should report when string can not be matched", func(t *testing.T) {
		_, found := EveNotificationTypeFromESIString("InvalidType")
		assert.False(t, found)
	})
	t.Run("can convert regular type to string", func(t *testing.T) {
		got, ok := st.EveNotificationTypeToESIString(app.StructureDestroyed)
		if assert.True(t, ok) {
			assert.Equal(t, "StructureDestroyed", got)
		}
	})
	t.Run("can report when type is irregular", func(t *testing.T) {
		_, ok := st.EveNotificationTypeToESIString(app.UnknownNotification)
		assert.False(t, ok)
	})
}
