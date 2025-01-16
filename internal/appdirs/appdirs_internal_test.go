package appdirs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppDirs(t *testing.T) {
	t.Run("can folder names", func(t *testing.T) {
		ad := AppDirs{
			Data:     "data",
			Log:      "log",
			Settings: "settings",
		}
		got := ad.Folders()
		expected := []string{"data", "log", "settings"}
		assert.ElementsMatch(t, expected, got)
	})
}
