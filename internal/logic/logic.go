// Package logic contains the app's business logic
package logic

import (
	"net/http"
	"time"

	"github.com/antihax/goesi"
)

var rl = requestLogger{}

var httpClient = &http.Client{
	Timeout:   time.Second * 30, // Timeout after 30 seconds
	Transport: rl,
}

var esiScopes = []string{
	"esi-characters.read_contacts.v1",
	"esi-mail.read_mail.v1",
	"esi-mail.organize_mail.v1",
	"esi-mail.send_mail.v1",
	"esi-search.search_structures.v1",
}

var esiClient = goesi.NewAPIClient(httpClient, "name@example.com")
