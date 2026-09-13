package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetSetting(ctx context.Context, key string) (string, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetSetting: %s: %w", key, err)
	}
	if key == "" {
		return "", wrapErr(app.ErrInvalid)
	}
	v, err := st.qRO.GetSetting(ctx, key)
	if err != nil {
		return "", wrapErr(convertGetError(err))
	}
	return v, nil
}

func (st *Storage) ListSettings(ctx context.Context) ([]queries.Setting, error) {
	rows, err := st.qRO.ListSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListSettings: %w", err)
	}
	return rows, nil
}

func (st *Storage) DeleteSetting(ctx context.Context, key string) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteSetting: %s: %w", key, err)
	}
	if key == "" {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteSetting(ctx, key)
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) SetSetting(ctx context.Context, key, value string) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("SetSetting: %s %s: %w", key, value, err)
	}
	if key == "" {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.SetSetting(ctx, queries.SetSettingParams{
		Key:   key,
		Value: value,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}
