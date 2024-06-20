package sso

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSOService(t *testing.T) {
	s := New("abc", http.DefaultClient, nil)
	assert.Equal(t, s.address(), "localhost:30123")
	assert.Equal(t, s.redirectURI(), "http://localhost:30123/callback")
	assert.Equal(t, s.CallbackPath, defaultSsoCallbackPath)
	assert.Equal(t, s.Port, defaultPort)
}
