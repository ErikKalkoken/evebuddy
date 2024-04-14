// Package service contains the business logic
package service

import (
	"net/http"
	"time"

	myHttp "example/evebuddy/internal/helper/http"
	"example/evebuddy/internal/repository"

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
	queries    *repository.Queries
}

func NewService(queries *repository.Queries) *Service {
	httpClient := &http.Client{
		Timeout:   time.Second * 30, // Timeout after 30 seconds
		Transport: myHttp.CustomTransport{},
	}
	esiClient := goesi.NewAPIClient(httpClient, "erik.kalkoken@gmail.com")
	s := Service{
		httpClient: httpClient,
		esiClient:  esiClient,
		queries:    queries,
	}
	return &s
}
