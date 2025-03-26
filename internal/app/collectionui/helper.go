package collectionui

import (
	"github.com/ErikKalkoken/evebuddy/internal/app"
)

func EntityNameOrFallback[T int | int32 | int64](e *app.EntityShort[T], fallback string) string {
	if e == nil {
		return fallback
	}
	return e.Name
}
