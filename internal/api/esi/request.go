package esi

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"example/esiapp/internal/cache"
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

func getESI(client *http.Client, path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, buildEsiUrl(path), nil)
	if err != nil {
		return nil, err
	}
	return sendRequestCached(client, req)
}

func postESI(client *http.Client, path string, data []byte) ([]byte, error) {
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
func sendRequest(client *http.Client, req *http.Request) ([]byte, string, error) {
	retry := 0
	for {
		slog.Info("Sending HTTP request", "method", req.Method, "url", req.URL)
		r, err := client.Do(req)
		if err != nil {
			return nil, "", err
		}
		if r.Body != nil {
			defer r.Body.Close()
		} else {
			return nil, "", nil
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, "", err
		}
		slog.Debug("ESI response", "body", string(body))
		if r.StatusCode == http.StatusOK {
			expires := r.Header.Get("Expires")
			return body, expires, nil
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
			return nil, "", fmt.Errorf("error %v", r.Status)
		}
		return nil, "", fmt.Errorf("error %v: %s", r.Status, e.Error)
	}
}

func sendRequestCached(client *http.Client, req *http.Request) ([]byte, error) {
	keyBase := fmt.Sprintf("%s-%s", req.URL.String(), req.Method)
	key := makeMD5Hash(keyBase)
	body, err := cache.Get(key)
	if err == nil {
		slog.Debug("Returning cached response", "key", keyBase)
		return body, nil
	}
	if err != cache.ErrCacheMiss {
		return nil, err
	}
	body, expires, err := sendRequest(client, req)
	if err != nil {
		return nil, err
	}
	expiresAt, err := time.Parse(time.RFC1123, expires)
	if err != nil {
		return body, nil
	}
	duration := time.Until(expiresAt)
	timeout := int(max(0, math.Floor(duration.Seconds())))
	cache.Set(key, body, timeout)
	return body, nil
}

func makeMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
