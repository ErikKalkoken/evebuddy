// Package janiceservice provides a service for accessing the Janice API.
package janiceservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	timeout = 5 * time.Second
	baseURL = "https://janice.e-351.com/api"
)

var ErrHttpError = errors.New("HTTP error")

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

// PricerItem represents a responce from the Pricer endpoint of the Janice API.
type PricerItem struct {
	Date   time.Time
	Market struct {
		ID   int
		Name string
	}
	BuyOrderCount     int32
	BuyVolume         int64
	SellOrderCount    int32
	SellVolume        int64
	ImmediatePrices   PricerItemValues
	Top5AveragePrices PricerItemValues
	ItemType          struct {
		EID            int64
		Name           string
		Volume         float64
		PackagedVolume float64
	}
}

// PricerItemValues represents a prices object within a PricerResponse.
type PricerItemValues struct {
	BuyPrice                 float64
	SplitPrice               float64
	SellPrice                float64
	BuyPrice5DayMedian       float64
	SplitPrice5DayMedianrice float64
	SellPrice5DayMedian      float64
	BuyPrice30DayMedian      float64
	SplitPrice30DayMedian    float64
	SellPrice30DayMedian     float64
}

func (s *JaniceService) FetchPrices(ctx context.Context, typeID int32) (PricerItem, error) {
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
		return info, nil
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("X-ApiKey", s.apiKey)
	r, err := s.httpClient.Do(req)
	if err != nil {
		return info, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return info, err
	}
	if r.StatusCode >= 400 {
		return info, fmt.Errorf("%s: %w", r.Status, ErrHttpError)
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	return info, nil
}
