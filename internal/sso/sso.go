// Package sso contains all logic for character authentication with the Eve Online SSO API.
// This package should not access any other internal packages, except helpers.
package sso

import memcache "github.com/ErikKalkoken/evebuddy/internal/cache"

const (
	host            = "localhost"
	port            = ":30123"
	address         = host + port
	oauthURL        = "https://login.eveonline.com/.well-known/oauth-authorization-server"
	ssoClientId     = "11ae857fe4d149b2be60d875649c05f1"
	ssoCallbackPath = "/callback"
	ssoHost         = "login.eveonline.com"
	ssoIssuer1      = "login.eveonline.com"
	ssoIssuer2      = "https://login.eveonline.com"
	ssoTokenUrl     = "https://login.eveonline.com/v2/oauth/token"
)

var cache = memcache.New()
