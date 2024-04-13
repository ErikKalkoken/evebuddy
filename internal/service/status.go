package service

import (
	"context"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"fyne.io/fyne/v2/data/binding"
)

func (s *Service) StartEsiStatusTicker(text binding.String) {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			s.updateESIStatus(text)
			<-ticker.C
		}
	}()
}

func (s *Service) updateESIStatus(text binding.String) error {
	status, resp, err := s.esiClient.ESI.StatusApi.GetStatus(context.Background(), nil)
	if err != nil {
		return err
	}
	isOffline := resp.StatusCode >= 500 || status.Players == 0
	var t string
	if isOffline {
		t = "OFFLINE"
	} else {
		p := message.NewPrinter(language.English)
		t = p.Sprintf("%d players", status.Players)
	}
	err = text.Set(t)
	if err != nil {
		return err
	}
	return nil
}
