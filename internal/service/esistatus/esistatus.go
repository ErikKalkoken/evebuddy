package esistatus

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/antihax/goesi"
)

type ESIStatus struct {
	esiClient *goesi.APIClient
}

func New(client *goesi.APIClient) *ESIStatus {
	es := &ESIStatus{esiClient: client}
	return es
}

type esiError struct {
	Error string `json:"error"`
}

func (s *ESIStatus) Fetch() (*model.ESIStatus, error) {
	ctx := context.Background()
	status, resp, err := s.esiClient.ESI.StatusApi.GetStatus(ctx, nil)
	if err != nil {
		return nil, err
	}
	x := &model.ESIStatus{PlayerCount: int(status.Players), StatusCode: resp.StatusCode}
	if resp.StatusCode >= 400 {
		var body []byte
		if resp.Body != nil {
			body, err = io.ReadAll(resp.Body)
			if err == nil {
				resp.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}
		var ee esiError
		if err := json.Unmarshal(body, &ee); err != nil {
			x.ErrorMessage = "Unknown error"
		} else {
			x.ErrorMessage = ee.Error
		}
	}
	return x, nil
}
