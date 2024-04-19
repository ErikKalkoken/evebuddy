package service

import (
	"context"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func (s *Service) FetchESIStatus() (string, error) {
	status, resp, err := s.esiClient.ESI.StatusApi.GetStatus(context.Background(), nil)
	if err != nil {
		return "", err
	}
	isOffline := resp.StatusCode >= 500 || status.Players == 0
	if isOffline {
		return "OFFLINE", nil
	}
	arg := message.NewPrinter(language.English)
	t := arg.Sprintf("%d players", status.Players)
	return t, nil
}
