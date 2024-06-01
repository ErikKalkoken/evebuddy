package esistatus

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/antihax/goesi"
	"github.com/antihax/goesi/esi"
)

type ESIStatus struct {
	esiClient *goesi.APIClient
}

func New(client *goesi.APIClient) *ESIStatus {
	es := &ESIStatus{esiClient: client}
	return es
}

func (s *ESIStatus) Fetch() (*model.ESIStatus, error) {
	ctx := context.Background()
	status, _, err := s.esiClient.ESI.StatusApi.GetStatus(ctx, nil)
	if err != nil {
		swaggerErr, ok := err.(esi.GenericSwaggerError)
		if ok {
			error := extractErrorMessage(swaggerErr)
			x := &model.ESIStatus{ErrorMessage: error}
			return x, nil
		} else {
			return nil, err
		}
	}
	x := &model.ESIStatus{PlayerCount: int(status.Players)}
	return x, nil
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
