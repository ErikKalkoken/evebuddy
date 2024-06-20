package mailrecipient_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/mailrecipient"
	"github.com/stretchr/testify/assert"
)

func TestNewRecipientsFromText(t *testing.T) {
	t.Run("can create from names", func(t *testing.T) {
		var cases = []struct {
			name string
			in   string
			out  string
		}{
			{"can create from text", "Erik Kalkoken [Character]", "Erik Kalkoken [Character]"},
			{"can create from text", "Erik Kalkoken", "Erik Kalkoken"},
			{"can create from text", "", ""},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				r := mailrecipient.NewFromText(tt.in)
				s := r.String()
				assert.Equal(t, tt.out, s)
			})
		}
	})
	t.Run("can create from empty text", func(t *testing.T) {
		r := mailrecipient.NewFromText("")
		assert.Equal(t, r.Size(), 0)
	})
}
