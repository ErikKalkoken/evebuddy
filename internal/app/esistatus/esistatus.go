// Package esistatus contains the ESI status service.
package esistatus

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
)

// Ensure service implements interface.
var _ app.ESIStatusService = (*ESIStatusService)(nil)

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
