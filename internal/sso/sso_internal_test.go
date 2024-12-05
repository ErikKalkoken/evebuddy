package sso

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSOService(t *testing.T) {
	t.Run("can create a new service", func(t *testing.T) {
		s := New("abc", http.DefaultClient, nil)
		assert.Equal(t, s.address(), "localhost:30123")
		assert.Equal(t, s.redirectURI(), "http://localhost:30123/callback")
		assert.Equal(t, s.CallbackPath, defaultSSOCallbackPath)
		assert.Equal(t, s.Port, defaultPort)
	})
	t.Run("can generate a correct start URL", func(t *testing.T) {
		got := makeStartURL("clientID", "challenge", "https://localhost/callback/", "state", []string{"esi-characters.read_blueprints.v1"})
		want := "https://login.eveonline.com/v2/oauth/authorize/?response_type=code&redirect_uri=https%3A%2F%2Flocalhost%2Fcallback%2F&client_id=clientID&scope=esi-characters.read_blueprints.v1&code_challenge=challenge&code_challenge_method=S256&state=state"
		u1, _ := url.Parse(got)
		u2, _ := url.Parse(want)
		assert.Equal(t, u2.Host, u1.Host)
		assert.Equal(t, u2.Path, u1.Path)
		q1, _ := url.ParseQuery(u1.RawQuery)
		q2, _ := url.ParseQuery(u2.RawQuery)
		assert.Equal(t, q2, q1)
	})
	t.Run("can generate code challenge", func(t *testing.T) {
		got, _ := calcCodeChallenge("abc")
		want := "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0"
		assert.Equal(t, want, got)
	})
}
