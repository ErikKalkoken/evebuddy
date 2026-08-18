package xgoesi

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

func TestExtractBodyForLog(t *testing.T) {
	t.Run("should return copy of the body", func(t *testing.T) {
		u, _ := url.Parse("http://www.example.com")
		r := &http.Response{
			Body: io.NopCloser(strings.NewReader("test")),
			Request: &http.Request{
				URL: u,
			},
		}
		x, err := extractBodyForLog(r)
		if assert.NoError(t, err) {
			assert.Equal(t, "test", x)
		}
	})
	t.Run("should return copy of the body as JSON", func(t *testing.T) {
		u, _ := url.Parse("http://www.example.com")
		r := &http.Response{
			Body: io.NopCloser(strings.NewReader("{\"alpha\": true}")),
			Request: &http.Request{
				URL: u,
			},
			Header: http.Header{headerContentTypeKey: []string{headerContentTypeJSON}},
		}
		x, err := extractBodyForLog(r)
		if assert.NoError(t, err) {
			assert.Equal(t, map[string]any{"alpha": true}, x)
		}
	})
	t.Run("should return empty when no body", func(t *testing.T) {
		u, _ := url.Parse("http://www.example.com")
		r := &http.Response{
			Request: &http.Request{
				URL: u,
			},
		}
		x, err := extractBodyForLog(r)
		if assert.NoError(t, err) {
			assert.Nil(t, x)
		}
	})
	t.Run("should redact blocked URL", func(t *testing.T) {
		u, _ := url.Parse("https://login.eveonline.com/v2/oauth/token")
		r := &http.Response{
			Body: io.NopCloser(strings.NewReader("test")),
			Request: &http.Request{
				URL: u,
			},
		}
		x, err := extractBodyForLog(r)
		if assert.NoError(t, err) {
			assert.Equal(t, "xxxxx", x)
		}
	})
	t.Run("should redact blocked URL", func(t *testing.T) {
		u, _ := url.Parse("https://login.eveonline.com/v2/oauth/token")
		r := &http.Response{
			Body: io.NopCloser(strings.NewReader("test")),
			Request: &http.Request{
				URL: u,
			},
			Header: http.Header{headerContentTypeKey: []string{"application/json; charset=UTF-8"}},
		}
		x, err := extractBodyForLog(r)
		if assert.NoError(t, err) {
			assert.Equal(t, map[string]bool(map[string]bool{"redacted": true}), x)
		}
	})
	t.Run("should return error", func(t *testing.T) {
		u, _ := url.Parse("http://www.example.com")
		b := io.NopCloser(iotest.ErrReader(errors.New("custom error")))
		r := &http.Response{
			Request: &http.Request{
				URL: u,
			},
			Body: b,
		}
		_, err := extractBodyForLog(r)
		assert.Error(t, err)
	})
}

func TestStatusText(t *testing.T) {
	t.Run("should return status text for normal codes", func(t *testing.T) {
		r := &http.Response{
			StatusCode: http.StatusOK,
		}
		x := statusText(r)
		assert.Equal(t, "200 OK", x)
	})
	t.Run("should return status text for 420", func(t *testing.T) {
		r := &http.Response{
			StatusCode: StatusTooManyErrors,
		}
		x := statusText(r)
		assert.Equal(t, "420 Error Limited", x)
	})
}
