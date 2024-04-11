// Package logic contains the app's business logic
package logic

import (
	"net/http"
	"time"

	myHttp "example/evebuddy/internal/helper/http"

	"github.com/antihax/goesi"
)

var httpClient = &http.Client{
	Timeout:   time.Second * 30, // Timeout after 30 seconds
	Transport: myHttp.CustomTransport{},
}

var esiScopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-search.search_structures.v1",
	"esi-wallet.read_character_wallet.v1",
}

var esiClient = goesi.NewAPIClient(httpClient, "name@example.com")
