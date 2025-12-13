// Package esistatusservice contains the ESI status service.
package esistatusservice

import (
	"context"
	"fmt"

	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
)

// ESIStatusService provides information about the current status of the ESI API.
type ESIStatusService struct {
	esiClient *goesi.APIClient
	sfg       *singleflight.Group
}

// New creates and returns a new instance of an ESI service.
func New(client *goesi.APIClient) *ESIStatusService {
	ess := &ESIStatusService{
		esiClient: client,
		sfg:       new(singleflight.Group),
	}
	return ess
}

func (ess *ESIStatusService) Fetch(ctx context.Context) (*app.ESIStatus, error) {
	x, err, _ := ess.sfg.Do("Fetch", func() (any, error) {
		ctx = xgoesi.NewContextWithOperationID(ctx, "GetStatus")
		status, _, err := ess.esiClient.ESI.StatusApi.GetStatus(ctx, nil)
		if err != nil {
			if swaggerErr, ok := err.(esi.GenericSwaggerError); ok {
				error := extractErrorMessage(swaggerErr)
				x := &app.ESIStatus{ErrorMessage: error}
				return x, nil
			}
			return nil, err
		}
		es := &app.ESIStatus{PlayerCount: int(status.Players)}
		return es, nil
	})
	if err != nil {
		return nil, err
	}
	return x.(*app.ESIStatus), nil
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

func (ess *ESIStatusService) DailyDowntime() string {
	const timeOnly = "15:04"
	start, finish := xgoesi.DailyDowntime()
	return fmt.Sprintf("%s - %s", start.Format(timeOnly), finish.Format(timeOnly))
}

func (ess *ESIStatusService) IsDailyDowntime() bool {
	return xgoesi.IsDailyDowntime()
}
