package xgoesi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/fnt-eve/goesi-openapi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

func TestRetryOn401(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	generate401error := func() error {
		const characterID = 42
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/implants", characterID),
			httpmock.NewJsonResponderOrPanic(http.StatusUnauthorized, map[string]any{
				"error": "unauthorized",
			}),
		)
		client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
			UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
		})
		_, _, err401 := client.ClonesAPI.GetCharactersCharacterIdImplants(t.Context(), characterID).Execute()
		return err401
	}

	t.Run("no retry when no error", func(t *testing.T) {
		// given
		var count int

		// when
		hasChanged, err := xgoesi.RetryOn401(1, "info", func() (bool, error) {
			count++
			return true, nil
		})

		// then
		assert.True(t, hasChanged)
		assert.NoError(t, err)
		xassert.Equal(t, 1, count)
	})

	t.Run("pass through error when not 401", func(t *testing.T) {
		// given
		var count int
		myErr := fmt.Errorf("my error")

		// when
		_, err := xgoesi.RetryOn401(1, "info", func() (bool, error) {
			count++
			return false, myErr
		})

		// then
		assert.ErrorIs(t, err, myErr)
		xassert.Equal(t, 1, count)
	})

	t.Run("should recover from 401", func(t *testing.T) {
		// given
		var count int

		// when
		hasChanged, err := xgoesi.RetryOn401(1, "info", func() (bool, error) {
			count++
			if count == 1 {
				return false, generate401error()
			}
			return true, nil
		})

		// then
		assert.True(t, hasChanged)
		assert.NoError(t, err)
		xassert.Equal(t, 2, count)
	})

	t.Run("should retry on 401 until max retries and then fail", func(t *testing.T) {
		// given
		var count int
		err401 := generate401error()

		// when
		_, err := xgoesi.RetryOn401(2, "info", func() (bool, error) {
			count++
			return false, err401
		})

		// then
		assert.ErrorIs(t, err, err401)
		xassert.Equal(t, 1+2, count)
	})
}
