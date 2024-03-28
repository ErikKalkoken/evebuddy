// Package sso contains all logic for character authentication with the Eve Online SSO API.
// This package should not access any other internal packages, except helpers.
package sso

import memcache "example/esiapp/internal/cache"

const (
	host            = "127.0.0.1"
	port            = ":8000"
	address         = host + port
	oauthURL        = "https://login.eveonline.com/.well-known/oauth-authorization-server"
	ssoClientId     = "882b6f0cbd4e44ad93aead900d07219b"
	ssoCallbackPath = "/sso/callback"
	ssoHost         = "login.eveonline.com"
	ssoIssuer1      = "login.eveonline.com"
	ssoIssuer2      = "https://login.eveonline.com"
	ssoTokenUrl     = "https://login.eveonline.com/v2/oauth/token"
)

var cache = memcache.New()
