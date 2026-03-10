// Package characterservice contains the EVE character service.
package characterservice

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
)

type Settings interface {
	MarketOrderRetentionDays() int
	MaxMails() int
	MaxWalletTransactions() int

	NotificationTypesEnabled() set.Set[string]
	NotifyCommunicationsEarliest() time.Time
	NotifyCommunicationsEnabled() bool
	NotifyContractsEarliest() time.Time
	NotifyContractsEnabled() bool
	NotifyMailsEarliest() time.Time
	NotifyMailsEnabled() bool
	NotifyPIEarliest() time.Time
	NotifyPIEnabled() bool
	NotifyTrainingEnabled() bool
}

type AuthClient interface {
	Authorize(ctx context.Context, scopes []string) (*eveauth.Token, error)
	RefreshToken(ctx context.Context, token *eveauth.Token) error
}

// Cache defines a cache.
type Cache interface {
	GetInt64(string) (int64, bool)
	SetInt64(string, int64, time.Duration)
	GetString(string) (string, bool)
	SetString(string, string, time.Duration)
	Delete(string)
}

// CharacterService provides access to all managed EVE Online characters both online and from local storage.
type CharacterService struct {
	authClient              AuthClient
	cache                   Cache
	concurrencyLimit        int
	ens                     *evenotification.EVENotificationService
	esiClient               *esi.APIClient
	eus                     *eveuniverseservice.EVEUniverseService
	httpClient              *http.Client
	scs                     *statuscache.StatusCache
	sendDesktopNotification func(title, content string) // Callback for sending a desktop notification via Fyne API
	settings                Settings
	sfg                     singleflight.Group
	sig                     *singleinstance.Group
	signals                 *app.Signals
	st                      *storage.Storage
}

type Params struct {
	AuthClient             AuthClient
	Cache                  Cache
	ConcurrencyLimit       int // max number of concurrent Goroutines (per group)
	ESIClient              *esi.APIClient
	EveNotificationService *evenotification.EVENotificationService
	EveUniverseService     *eveuniverseservice.EVEUniverseService
	Settings               Settings
	Signals                *app.Signals
	StatusCacheService     *statuscache.StatusCache
	Storage                *storage.Storage
	// optional
	HTTPClient              *http.Client
	SendDesktopNotification func(title, content string)
}

// New creates a new character service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(arg Params) *CharacterService {
	if arg.AuthClient == nil {
		panic("AuthClient missing")
	}
	if arg.Cache == nil {
		panic("Cache missing")
	}
	if arg.ESIClient == nil {
		panic("ESIClient missing")
	}
	if arg.EveNotificationService == nil {
		panic("EveNotificationService missing")
	}
	if arg.EveUniverseService == nil {
		panic("EveUniverseService missing")
	}
	if arg.Settings == nil {
		panic("Settings missing")
	}
	if arg.Signals == nil {
		panic("Signals missing")
	}
	if arg.StatusCacheService == nil {
		panic("StatusCacheService missing")
	}
	if arg.Storage == nil {
		panic("Storage missing")
	}
	s := &CharacterService{
		authClient:       arg.AuthClient,
		cache:            arg.Cache,
		concurrencyLimit: -1, // Default is no limit
		ens:              arg.EveNotificationService,
		esiClient:        arg.ESIClient,
		eus:              arg.EveUniverseService,
		scs:              arg.StatusCacheService,
		sendDesktopNotification: func(_, _ string) {
			slog.Warn("Desktop notifications not configured")
		},
		settings: arg.Settings,
		signals:  arg.Signals,
		sig:      singleinstance.NewGroup(),
		st:       arg.Storage,
	}
	if arg.HTTPClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = arg.HTTPClient
	}
	if arg.ConcurrencyLimit > 0 {
		s.concurrencyLimit = arg.ConcurrencyLimit
	}
	if arg.SendDesktopNotification != nil {
		s.sendDesktopNotification = arg.SendDesktopNotification
	}
	return s
}

// DumpData returns the current content of the given SQL tables as JSON string.
// When no tables are given all tables will be included.
// This is a low-level method meant mainly for debugging.
func (s *CharacterService) DumpData(tables ...string) string {
	return s.st.DumpData(tables...)
}
