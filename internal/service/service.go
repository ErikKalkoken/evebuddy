// Package service contains the business logic
package service

import (
	"net/http"
	"time"

	myHttp "example/evebuddy/internal/helper/http"
	"example/evebuddy/internal/storage"

	"github.com/antihax/goesi"
)

var esiScopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-search.search_structures.v1",
	"esi-skills.read_skills.v1",
	"esi-wallet.read_character_wallet.v1",
}

type Service struct {
	httpClient *http.Client
	esiClient  *goesi.APIClient
	r          *storage.Storage
}

func NewService(r *storage.Storage) *Service {
	httpClient := &http.Client{
		Timeout:   time.Second * 30, // Timeout after 30 seconds
		Transport: myHttp.CustomTransport{},
	}
	userAgent := "EveBuddy kalkoken87@gmail.com"
	esiClient := goesi.NewAPIClient(httpClient, userAgent)
	s := Service{
		httpClient: httpClient,
		esiClient:  esiClient,
		r:          r,
	}
	return &s
}
