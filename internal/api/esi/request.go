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
	statusCode int
}

func getESI(client *http.Client, path string) (*esiResponse, error) {
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
	return sendRequestCached(client, req)
}

func buildEsiUrl(path string) string {
	u := fmt.Sprintf("%s%s", esiBaseUrl, path)
	return u
}

// TODO: retry also on timeouts
func sendRequest(client *http.Client, req *http.Request) (*esiResponse, error) {
	retry := 0
	for {
		slog.Info("Sending HTTP request", "method", req.Method, "url", req.URL)
		r, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		res := &esiResponse{statusCode: r.StatusCode, header: r.Header}
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
		slog.Debug("ESI response", "body", string(body))
		if r.StatusCode == http.StatusOK {
			return res, nil
		}
		slog.Warn("ESI status response not OK", "status", r.Status)
		if r.StatusCode == http.StatusBadGateway || r.StatusCode == http.StatusGatewayTimeout || r.StatusCode == http.StatusServiceUnavailable {
			if retry < EsiMaxRetries {
				retry++
				slog.Info("Retrying", "retry", retry, "maxRetries", EsiMaxRetries)
				wait := time.Millisecond * time.Duration(200*retry)
				time.Sleep(wait)
				continue
			}
		}
		var e esiError
		if err := json.Unmarshal(body, &e); err != nil {
			return nil, fmt.Errorf("error %v", r.Status)
		}
		return nil, fmt.Errorf("error %v: %s", r.Status, e.Error)
	}
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
