package sso

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestToken(t *testing.T) {
	var claims jwt.MapClaims = map[string]any{
		"scp": []any{
			"esi-skills.read_skills.v1",
			"esi-skills.read_skillqueue.v1",
		},
		"jti":    "998e12c7-3241-43c5-8355-2c48822e0a1b",
		"kid":    "JWT-Signature-Key",
		"sub":    "CHARACTER:EVE:123123",
		"azp":    ssoClientId,
		"tenant": "tranquility",
		"tier":   "live",
		"region": "world",
		"aud": []any{
			ssoClientId,
			"EVE Online",
		},
		"name":  "Some Bloke",
		"owner": "8PmzCeTKb4VFUDrHLc/AeZXDSWM=",
		"exp":   1648563218,
		"iat":   1648562018,
		"iss":   "login.eveonline.com",
	}
	t.Run("can create new token", func(t *testing.T) {
		tp := &tokenPayload{
			AccessToken:  "access token",
			RefreshToken: "refresh token",
			TokenType:    "token type",
			ExpiresIn:    1800,
		}
		o, err := newToken(tp, claims)
		if assert.NoError(t, err) {
			assert.Equal(t, "access token", o.AccessToken)
			assert.Equal(t, "refresh token", o.RefreshToken)
			assert.Equal(t, "token type", o.TokenType)
			assert.Equal(t, int32(123123), o.CharacterID)
			assert.Equal(t, "Some Bloke", o.CharacterName)
			assert.Equal(t, []string{"esi-skills.read_skills.v1", "esi-skills.read_skillqueue.v1"}, o.Scopes)
			assert.WithinDuration(t, time.Now().Add(1800*time.Second), o.ExpiresAt, 10*time.Second)
		}
	})
	t.Run("can calculate expires at", func(t *testing.T) {
		tp := tokenPayload{
			ExpiresIn: 1800,
		}
		got := tp.expiresAt()
		assert.WithinDuration(t, time.Now().Add(1800*time.Second), got, 10*time.Second)
	})
}

func TestValidateClaims(t *testing.T) {
	t.Run("should not report error when token is valid", func(t *testing.T) {
		var claims jwt.MapClaims = map[string]any{
			"scp": []any{
				"esi-skills.read_skills.v1",
				"esi-skills.read_skillqueue.v1",
			},
			"jti":    "998e12c7-3241-43c5-8355-2c48822e0a1b",
			"kid":    "JWT-Signature-Key",
			"sub":    "CHARACTER:EVE:123123",
			"azp":    ssoClientId,
			"tenant": "tranquility",
			"tier":   "live",
			"region": "world",
			"aud": []any{
				ssoClientId,
				"EVE Online",
			},
			"name":  "Some Bloke",
			"owner": "8PmzCeTKb4VFUDrHLc/AeZXDSWM=",
			"exp":   1648563218,
			"iat":   1648562018,
			"iss":   "login.eveonline.com",
		}
		assert.NoError(t, validateClaims(claims))
	})
	t.Run("should not report error when token is valid 2", func(t *testing.T) {
		var claims jwt.MapClaims = map[string]any{
			"scp": []any{
				"esi-skills.read_skills.v1",
				"esi-skills.read_skillqueue.v1",
			},
			"jti":    "998e12c7-3241-43c5-8355-2c48822e0a1b",
			"kid":    "JWT-Signature-Key",
			"sub":    "CHARACTER:EVE:123123",
			"azp":    ssoClientId,
			"tenant": "tranquility",
			"tier":   "live",
			"region": "world",
			"aud": []any{
				ssoClientId,
				"EVE Online",
			},
			"name":  "Some Bloke",
			"owner": "8PmzCeTKb4VFUDrHLc/AeZXDSWM=",
			"exp":   1648563218,
			"iat":   1648562018,
			"iss":   "https://login.eveonline.com",
		}
		assert.NoError(t, validateClaims(claims))
	})
	t.Run("should abort when audience is wrong", func(t *testing.T) {
		var claims jwt.MapClaims = map[string]any{
			"scp": []any{
				"esi-skills.read_skills.v1",
				"esi-skills.read_skillqueue.v1",
			},
			"jti":    "998e12c7-3241-43c5-8355-2c48822e0a1b",
			"kid":    "JWT-Signature-Key",
			"sub":    "CHARACTER:EVE:123123",
			"azp":    "wrongClientID",
			"tenant": "tranquility",
			"tier":   "live",
			"region": "world",
			"aud": []any{
				"wrongClientID",
				"EVE Online",
			},
			"name":  "Some Bloke",
			"owner": "8PmzCeTKb4VFUDrHLc/AeZXDSWM=",
			"exp":   1648563218,
			"iat":   1648562018,
			"iss":   "login.eveonline.com",
		}
		assert.Error(t, validateClaims(claims))
	})
	t.Run("should abort when audience is wrong 2", func(t *testing.T) {
		var claims jwt.MapClaims = map[string]any{
			"scp": []any{
				"esi-skills.read_skills.v1",
				"esi-skills.read_skillqueue.v1",
			},
			"jti":    "998e12c7-3241-43c5-8355-2c48822e0a1b",
			"kid":    "JWT-Signature-Key",
			"sub":    "CHARACTER:EVE:123123",
			"azp":    ssoClientId,
			"tenant": "tranquility",
			"tier":   "live",
			"region": "world",
			"aud": []any{
				ssoClientId,
				"wrong issuer",
			},
			"name":  "Some Bloke",
			"owner": "8PmzCeTKb4VFUDrHLc/AeZXDSWM=",
			"exp":   1648563218,
			"iat":   1648562018,
			"iss":   "login.eveonline.com",
		}
		assert.Error(t, validateClaims(claims))
	})
	t.Run("should not report error when issuer is wrong", func(t *testing.T) {
		var claims jwt.MapClaims = map[string]any{
			"scp": []any{
				"esi-skills.read_skills.v1",
				"esi-skills.read_skillqueue.v1",
			},
			"jti":    "998e12c7-3241-43c5-8355-2c48822e0a1b",
			"kid":    "JWT-Signature-Key",
			"sub":    "CHARACTER:EVE:123123",
			"azp":    ssoClientId,
			"tenant": "tranquility",
			"tier":   "live",
			"region": "world",
			"aud": []any{
				ssoClientId,
				"EVE Online",
			},
			"name":  "Some Bloke",
			"owner": "8PmzCeTKb4VFUDrHLc/AeZXDSWM=",
			"exp":   1648563218,
			"iat":   1648562018,
			"iss":   "login.eve2online.com",
		}
		assert.Error(t, validateClaims(claims))
	})
}
