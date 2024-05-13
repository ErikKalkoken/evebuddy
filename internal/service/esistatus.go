package service

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func (s *Service) FetchESIStatus() (*model.ESIStatus, error) {
	status, resp, err := s.esiClient.ESI.StatusApi.GetStatus(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	isOnline := resp.StatusCode < 500 || status.Players == 0
	x := &model.ESIStatus{IsOnline: isOnline, PlayerCount: int(status.Players)}
	return x, nil
}
