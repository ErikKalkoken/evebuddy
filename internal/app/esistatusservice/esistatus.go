// Package esistatusservice contains the ESI status service.
package esistatusservice

import (
	"context"
	"fmt"
	"time"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

// daily downtime
const (
	dailyDowntimeStart  = "11h0m"
	dailyDowntimeFinish = "11h15m"
)

// ESIStatusService provides information about the current status of the ESI API.
type ESIStatusService struct {
	esiClient *goesi.APIClient
}

// New creates and returns a new instance of an ESI service.
func New(client *goesi.APIClient) *ESIStatusService {
	ess := &ESIStatusService{esiClient: client}
	return ess
}

func (ess *ESIStatusService) Fetch(ctx context.Context) (*app.ESIStatus, error) {
	ctx = xesi.NewContextWithOperationID(ctx, "GetStatus")
	status, _, err := ess.esiClient.ESI.StatusApi.GetStatus(ctx, nil)
	if err != nil {
		swaggerErr, ok := err.(esi.GenericSwaggerError)
		if ok {
			error := extractErrorMessage(swaggerErr)
			x := &app.ESIStatus{ErrorMessage: error}
			return x, nil
		} else {
			return nil, err
		}
	}
	es := &app.ESIStatus{PlayerCount: int(status.Players)}
	return es, nil
}

func extractErrorMessage(err esi.GenericSwaggerError) string {
	var detail string
	switch t2 := err.Model().(type) {
	case esi.BadRequest:
		detail = t2.Error_
	case esi.ErrorLimited:
		detail = t2.Error_
	case esi.GatewayTimeout:
		detail = t2.Error_
	case esi.InternalServerError:
		detail = t2.Error_
	case esi.ServiceUnavailable:
		detail = t2.Error_
	default:
		detail = "general swagger error"
	}
	return fmt.Sprintf("%s: %s", err.Error(), detail)
}

// DailyDowntime returns the daily downtime period.
func (ess *ESIStatusService) DailyDowntime() string {
	const timeOnly = "15:04"
	start, finish := calcDowntimePeriod(time.Now(), dailyDowntimeStart, dailyDowntimeFinish)
	return fmt.Sprintf("%s - %s", start.Format(timeOnly), finish.Format(timeOnly))
}

// IsDailyDowntime reports whether the daily downtime is currently planned to happen.
func (ess *ESIStatusService) IsDailyDowntime() bool {
	return isDailyDowntime(dailyDowntimeStart, dailyDowntimeFinish, time.Now())
}

func isDailyDowntime(startStr, finishStr string, t time.Time) bool {
	start, finish := calcDowntimePeriod(t, startStr, finishStr)
	if t.Before(start) {
		return false
	}
	if t.After(finish) {
		return false
	}
	return true
}

func calcDowntimePeriod(t time.Time, startStr string, finishStr string) (time.Time, time.Time) {
	today0 := t.UTC().Truncate(24 * time.Hour)
	d1, err := time.ParseDuration(startStr)
	if err != nil {
		panic(err)
	}
	start := today0.Add(d1)
	d2, err := time.ParseDuration(finishStr)
	if err != nil {
		panic(err)
	}
	finish := today0.Add(d2)
	return start, finish
}
