package esistatusservice_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	es := esistatusservice.New(client)
	ctx := context.TODO()
	t.Run("should return full report when ESI is online", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/status/",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"players":        12345,
				"server_version": "1132976",
				"start_time":     "2017-01-02T12:34:56Z",
			}))
		// when
		got, err := es.Fetch(ctx)
		// then
		if assert.NoError(t, err) {
			want := &app.ESIStatus{
				PlayerCount:  12345,
				ErrorMessage: "",
			}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return general error message when ESI returns unexpected error code", func(t *testing.T) {
		// given
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/status/",
			httpmock.NewJsonResponderOrPanic(418, map[string]any{
				"error": "custom error message",
			}))
		// when
		got, err := es.Fetch(ctx)
		// then
		if assert.NoError(t, err) {
			want := &app.ESIStatus{
				ErrorMessage: "418 I'm a teapot: general swagger error",
			}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return error when a technical error occurred", func(t *testing.T) {
		// given
		myErr := errors.New("my error")
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/status/",
			httpmock.NewErrorResponder(myErr))
		// when
		_, err := es.Fetch(ctx)
		// then
		assert.ErrorIs(t, err, myErr)
	})
}

func TestFetchSwaggerErrors(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	es := esistatusservice.New(client)
	ctx := context.TODO()
	statusCodes := []int{400, 420, 500, 503, 504}
	for _, code := range statusCodes {
		t.Run(fmt.Sprintf("should return extracted error message when ESI returns status %d", code), func(t *testing.T) {
			// given
			httpmock.Reset()
			httpmock.RegisterResponder(
				"GET",
				"https://esi.evetech.net/v1/status/",
				httpmock.NewJsonResponderOrPanic(code, map[string]any{
					"error": "custom error message",
				}))
			// when
			got, err := es.Fetch(ctx)
			// then
			if assert.NoError(t, err) {
				assert.True(t, strings.HasPrefix(got.ErrorMessage, fmt.Sprint(code)))
			}
		})
	}
}
