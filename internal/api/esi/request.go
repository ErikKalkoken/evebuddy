package esi

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"time"
)

const EsiMaxRetries = 3

// A generic error returned from ESI
type esiError struct {
	Error string `json:"error"`
}

type esiResponse struct {
	body       []byte
	header     http.Header
	status     string
	statusCode int
	errorText  string
}

func (r esiResponse) ok() bool {
	return r.statusCode < 400
}

func (r esiResponse) error() error {
	if r.ok() {
		return nil
	}
	return fmt.Errorf("ESI error: %s: %s", r.status, r.errorText)
}

// getESI sends a GET request to ESI and returns the response.
// HTTP error codes are returned as error values.
func getESI(client *http.Client, path string) (*esiResponse, error) {
	res, err := getESIWithStatus(client, path)
	if err != nil {
		return nil, err
	}
	if !res.ok() {
		return nil, res.error()
	}
	return res, nil
}

// getESI sends a GET request to ESI and returns the response.
// HTTP error codes are not returned in the esiResponse.
func getESIWithStatus(client *http.Client, path string) (*esiResponse, error) {
	req, err := http.NewRequest(http.MethodGet, buildEsiUrl(path), nil)
	if err != nil {
		return nil, err
	}
	return sendRequestCached(client, req)
}

func postESI(client *http.Client, path string, data []byte) (*esiResponse, error) {
	req, err := http.NewRequest(http.MethodPost, buildEsiUrl(path), bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := sendRequest(client, req)
	if err != nil {
		return nil, err
	}
	if !res.ok() {
		return nil, res.error()
	}
	return res, nil
}

func buildEsiUrl(path string) string {
	u := fmt.Sprintf("%s%s", esiBaseUrl, path)
	return u
}

// TODO: retry also on timeouts
func sendRequest(client *http.Client, req *http.Request) (*esiResponse, error) {
	retry := 0
	for {
		slog.Info("HTTP request", "method", req.Method, "url", removeToken(*req.URL))
		r, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		res := &esiResponse{status: r.Status, statusCode: r.StatusCode, header: r.Header}
		if r.Body != nil {
			defer r.Body.Close()
		} else {
			return res, nil
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		res.body = body
		slog.Debug("ESI response", "status", res.status, "body", string(res.body), "header", res.header)
		if r.StatusCode == http.StatusOK {
			return res, nil
		}
		var e esiError
		if err := json.Unmarshal(body, &e); err == nil {
			res.errorText = e.Error
		}
		slog.Warn("ESI error", "error", res.error())
		if r.StatusCode == http.StatusBadGateway || r.StatusCode == http.StatusGatewayTimeout || r.StatusCode == http.StatusServiceUnavailable {
			if retry < EsiMaxRetries {
				retry++
				slog.Info("Retrying", "retry", retry, "maxRetries", EsiMaxRetries)
				wait := time.Millisecond * time.Duration(200*retry)
				time.Sleep(wait)
				continue
			}
		}
		return res, nil
	}
}

func removeToken(u url.URL) string {
	v, _ := url.ParseQuery(u.RawQuery)
	v.Del("token")
	u.RawQuery = v.Encode()
	return u.String()
}

func sendRequestCached(client *http.Client, req *http.Request) (*esiResponse, error) {
	keyBase := fmt.Sprintf("%s-%s", req.URL.String(), req.Method)
	key := makeMD5Hash(keyBase)
	v, found := cache.Get(key)
	if found {
		slog.Debug("Returning cached response", "key", keyBase)
		res := v.(esiResponse)
		return &res, nil
	}
	res, err := sendRequest(client, req)
	if err != nil {
		return nil, err
	}
	expires := res.header.Get("Expires")
	if expires == "" {
		return res, nil
	}
	expiresAt, err := time.Parse(time.RFC1123, expires)
	if err != nil {
		return res, nil
	}
	duration := time.Until(expiresAt)
	timeout := int(max(0, math.Floor(duration.Seconds())))
	cache.Set(key, *res, timeout)
	return res, nil
}

func makeMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}