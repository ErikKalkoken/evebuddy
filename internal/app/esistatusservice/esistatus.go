// Package esistatusservice contains the ESI status service.
package esistatusservice

import (
	"context"

	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

// ESIStatusService provides information about the current status of the ESI API.
type ESIStatusService struct {
	esiClient *esi.APIClient
	sfg       singleflight.Group
}

// New creates and returns a new instance of an ESI service.
func New(client *esi.APIClient) *ESIStatusService {
	ess := &ESIStatusService{
		esiClient: client,
	}
	return ess
}

// Fetch retrieves an update from ESI and returns it.
func (s *ESIStatusService) Fetch(ctx context.Context) (*app.ESIStatus, error) {
	o, err, _ := xsingleflight.Do(&s.sfg, "Fetch", func() (*app.ESIStatus, error) {
		ctx = xgoesi.NewContextWithOperationID(ctx, "GetStatus")
		status, response, err := s.esiClient.StatusAPI.GetStatus(ctx).Execute()
		if err != nil {
			if swaggerErr, ok := err.(*esi.GenericOpenAPIError); ok {
				msg := swaggerErr.Error()
				if x, ok := swaggerErr.Model().(esi.Error); ok {
					msg += ": " + x.Error
				}
				return &app.ESIStatus{ErrorMessage: msg, HTTPStatusCode: response.StatusCode}, nil
			}
			return nil, err
		}
		es := &app.ESIStatus{HTTPStatusCode: response.StatusCode, PlayerCount: int(status.Players)}
		return es, nil
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

// func extractErrorMessage(err esi.GenericOpenAPIError) string {
// 	var detail string
// 	switch t2 := err.Model().(type) {
// 	case esi.model:
// 		detail = t2.Error_
// 	case esi.ErrorLimited:
// 		detail = t2.Error_
// 	case esi.GatewayTimeout:
// 		detail = t2.Error_
// 	case esi.InternalServerError:
// 		detail = t2.Error_
// 	case esi.ServiceUnavailable:
// 		detail = t2.Error_
// 	default:
// 		detail = "general swagger error"
// 	}
// 	return fmt.Sprintf("%s: %s", err.Error(), detail)
// }

// IsDailyDowntime reports whether the daily downtime is currently planned to happen.
func (s *ESIStatusService) IsDailyDowntime() bool {
	return xgoesi.IsDailyDowntime()
}
