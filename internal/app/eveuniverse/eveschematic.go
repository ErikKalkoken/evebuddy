package eveuniverse

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func (eu *EveUniverseService) GetOrCreateSchematicESI(ctx context.Context, id int32) (*app.EveSchematic, error) {
	x, err := eu.st.GetEveSchematic(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveSchematicFromESI(ctx, id)
	} else if err != nil {
		return x, err
	}
	return x, nil
}

func (eu *EveUniverseService) createEveSchematicFromESI(ctx context.Context, id int32) (*app.EveSchematic, error) {
	key := fmt.Sprintf("createEveSchematicFromESI-%d", id)
	y, err, _ := eu.sfg.Do(key, func() (any, error) {
		r, _, err := eu.esiClient.ESI.PlanetaryInteractionApi.GetUniverseSchematicsSchematicId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveSchematicParams{
			ID:        id,
			CycleTime: int(r.CycleTime),
			Name:      r.SchematicName,
		}
		return eu.st.CreateEveSchematic(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return y.(*app.EveSchematic), nil
}
