package eveuniverseservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (s *EveUniverseService) GetOrCreateSchematicESI(ctx context.Context, id int32) (*app.EveSchematic, error) {
	o, err := s.st.GetEveSchematic(ctx, id)
	if errors.Is(err, app.ErrNotFound) {
		return s.createSchematicFromESI(ctx, id)
	}
	return o, err
}

func (s *EveUniverseService) createSchematicFromESI(ctx context.Context, id int32) (*app.EveSchematic, error) {
	key := fmt.Sprintf("createSchematicFromESI-%d", id)
	y, err, _ := s.sfg.Do(key, func() (any, error) {
		r, _, err := s.esiClient.ESI.PlanetaryInteractionApi.GetUniverseSchematicsSchematicId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSchematicParams{
			ID:        id,
			CycleTime: int(r.CycleTime),
			Name:      r.SchematicName,
		}
		return s.st.CreateEveSchematic(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveSchematic), nil
}
