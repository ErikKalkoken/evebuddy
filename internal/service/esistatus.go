package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type ESIError struct {
	Error string `json:"error"`
}

func (s *Service) FetchESIStatus() (*model.ESIStatus, error) {
	status, resp, err := s.esiClient.ESI.StatusApi.GetStatus(context.Background(), nil)
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
		var ee ESIError
		if err := json.Unmarshal(body, &ee); err != nil {
			x.ErrorMessage = "Unknown error"
		} else {
			x.ErrorMessage = ee.Error
		}
	}
	return x, nil
}
