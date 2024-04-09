package logic

import (
	"context"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"fyne.io/fyne/v2/data/binding"
)

func StartEsiStatusTicker(text binding.String) {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for {
			updateESIStatus(text)
			<-ticker.C
		}
	}()
}

func updateESIStatus(text binding.String) error {
	status, resp, err := esiClient.ESI.StatusApi.GetStatus(context.Background(), nil)
	if err != nil {
		return err
	}
	isOffline := resp.StatusCode >= 500
	var s string
	if isOffline {
		s = "OFFLINE"
	} else {
		p := message.NewPrinter(language.English)
		s = p.Sprintf("%d players", status.Players)
	}
	err = text.Set(s)
	if err != nil {
		return err
	}
	return nil
}
