package esistatus_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/esistatus"
	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestXxx(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	client := goesi.NewAPIClient(nil, "")
	es := esistatus.New(client)
	t.Run("should return status when ESI is online", func(t *testing.T) {
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
		got, err := es.Fetch()
		// then
		if assert.NoError(t, err) {
			want := &model.ESIStatus{
				StatusCode:   200,
				PlayerCount:  12345,
				ErrorMessage: "",
			}
			assert.Equal(t, want, got)
		}
	})

}
