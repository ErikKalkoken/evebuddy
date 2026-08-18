// Package janiceservice provides a service for accessing the Janice API.
package janiceservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	timeout = 5 * time.Second
	baseURL = "https://janice.e-351.com/api"
)

var ErrHTTPError = errors.New("HTTP error")

type JaniceService struct {
	apiKey     string
	httpClient *http.Client
}

func New(httpClient *http.Client, apiKey string) *JaniceService {
	if httpClient == nil {
		panic("need HTTP client")
	}
	s := &JaniceService{
		httpClient: httpClient,
		apiKey:     apiKey,
	}
	return s
}

// PricerItem represents a response from the Pricer endpoint of the Janice API.
type PricerItem struct {
	Date   time.Time `json:"date"`
	Market struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"market"`
	BuyOrderCount     int64            `json:"buyOrderCount"`
	BuyVolume         int64            `json:"buyVolume"`
	SellOrderCount    int64            `json:"sellOrderCount"`
	SellVolume        int64            `json:"sellVolume"`
	ImmediatePrices   PricerItemValues `json:"immediatePrices"`
	Top5AveragePrices PricerItemValues `json:"top5AveragePrices"`
	ItemType          struct {
		EID            int64   `json:"eid"`
		Name           string  `json:"name"`
		Volume         float64 `json:"volume"`
		PackagedVolume float64 `json:"packagedVolume"`
	} `json:"itemType"`
}

// PricerItemValues represents a prices object within a PricerResponse.
type PricerItemValues struct {
	BuyPrice              float64 `json:"buyPrice"`
	SplitPrice            float64 `json:"splitPrice"`
	SellPrice             float64 `json:"sellPrice"`
	BuyPrice5DayMedian    float64 `json:"buyPrice5DayMedian"`
	SplitPrice5DayMedian  float64 `json:"splitPrice5DayMedian"`
	SellPrice5DayMedian   float64 `json:"sellPrice5DayMedian"`
	BuyPrice30DayMedian   float64 `json:"buyPrice30DayMedian"`
	SplitPrice30DayMedian float64 `json:"splitPrice30DayMedian"`
	SellPrice30DayMedian  float64 `json:"sellPrice30DayMedian"`
}

func (s *JaniceService) FetchPrices(ctx context.Context, typeID int64) (PricerItem, error) {
	var info PricerItem
	if typeID <= 0 {
		return info, errors.New("invalid typeID")
	}
	if s.apiKey == "" {
		return info, errors.New("missing API key")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/rest/v2/pricer/%d", baseURL, typeID), nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("X-ApiKey", s.apiKey)
	r, err := s.httpClient.Do(req)
	if err != nil {
		return info, err
	}
	defer r.Body.Close()
	if r.StatusCode >= 400 {
		var data any
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			slog.Warn("Error response from Janice was not JSON", "error", err)
		} else {
			slog.Warn("Error response from Janice", "typeID", typeID, "response", data)
		}
		return info, fmt.Errorf("%s: %w", r.Status, ErrHTTPError)
	}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		return info, err
	}
	return info, nil
}
