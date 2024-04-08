// Package logic contains the app's business logic
package logic

import (
	"bytes"
	"io"
	"log/slog"
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

type requestLogger struct{}

func (r requestLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	bodyStr := ""
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err == nil {
			bodyStr = string(body)
			req.Body = io.NopCloser(bytes.NewBuffer(body))
		}
	}
	slog.Info("HTTP request", "method", req.Method, "url", req.URL, "body", bodyStr)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	return resp, err
}
