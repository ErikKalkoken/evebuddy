package esi

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type myData struct {
	Alpha int `json:"alpha"`
}

func TestUnmarshalResponse(t *testing.T) {
	t.Run("should return data", func(t *testing.T) {
		// given
		json := `{"alpha": 42}`
		resp := http.Response{Body: io.NopCloser(strings.NewReader(json))}
		// when
		o, err := unmarshalResponse[myData](&resp)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 42, o.Alpha)
		}
	})
	t.Run("should return zero values when Body is empty", func(t *testing.T) {
		// given
		resp := http.Response{Body: nil}
		// when
		o, err := unmarshalResponse[myData](&resp)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 0, o.Alpha)
		}
	})
	t.Run("should return error when body can not be marshalled", func(t *testing.T) {
		// given
		json := `{"alpha": "wrong"}`
		resp := http.Response{Body: io.NopCloser(strings.NewReader(json))}
		// when
		o, err := unmarshalResponse[myData](&resp)
		// then
		if assert.Error(t, err) {
			assert.Equal(t, 0, o.Alpha)
		}
	})
}
