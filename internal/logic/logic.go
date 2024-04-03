// Package logic contains the app's business logic
package logic

import (
	"net/http"
	"time"
)

var httpClient = &http.Client{
	Timeout: time.Second * 30, // Timeout after 30 seconds
}
