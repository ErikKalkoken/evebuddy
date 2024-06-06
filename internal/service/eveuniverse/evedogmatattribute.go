package eveuniverse

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func (eu *EveUniverseService) GetEveDogmaAttribute(ctx context.Context, id int32) (*model.EveDogmaAttribute, error) {
	o, err := eu.st.GetEveDogmaAttribute(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return o, nil
}

func (eu *EveUniverseService) GetOrCreateEveDogmaAttributeESI(ctx context.Context, id int32) (*model.EveDogmaAttribute, error) {
	o, err := eu.st.GetEveDogmaAttribute(ctx, id)
	if errors.Is(err, storage.ErrNotFound) {
		return eu.createEveDogmaAttributeFromESI(ctx, id)
	} else if err != nil {
		return o, err
	}
	return o, nil
}

func (eu *EveUniverseService) createEveDogmaAttributeFromESI(ctx context.Context, id int32) (*model.EveDogmaAttribute, error) {
	key := fmt.Sprintf("createEveDogmaAttributeFromESI-%d", id)
	x, err, _ := eu.sfg.Do(key, func() (any, error) {
		o, _, err := eu.esiClient.ESI.DogmaApi.GetDogmaAttributesAttributeId(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		arg := storage.CreateEveDogmaAttributeParams{
			ID:           o.AttributeId,
			DefaultValue: o.DefaultValue,
			Description:  o.Description,
			DisplayName:  o.DisplayName,
			IconID:       o.IconId,
			Name:         o.Name,
			IsHighGood:   o.HighIsGood,
			IsPublished:  o.Published,
			IsStackable:  o.Stackable,
			UnitID:       o.UnitId,
		}
		return eu.st.CreateEveDogmaAttribute(ctx, arg)
	})
	if err != nil {
		return nil, err
	}
	return x.(*model.EveDogmaAttribute), nil
}
