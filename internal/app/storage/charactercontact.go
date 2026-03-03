package storage

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) DeleteCharacterContacts(ctx context.Context, characterID int64, contactIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterContacts for character %d and contact IDs: %v: %w", characterID, contactIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if contactIDs.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterContacts(ctx, queries.DeleteCharacterContactsParams{
		CharacterID: characterID,
		ContactIds:  slices.Collect(contactIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Contacts deleted", "characterID", characterID, "contactIDs", contactIDs)
	return nil
}

func (st *Storage) GetCharacterContact(ctx context.Context, characterID int64, contactID int64) (*app.CharacterContact, error) {
	r, err := st.qRO.GetCharacterContact(ctx, queries.GetCharacterContactParams{
		CharacterID: characterID,
		ContactID:   contactID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetCharacterContact for character %d: %w", characterID, convertGetError(err))
	}
	o := characterContactFromDBModel(r.CharacterContact, r.EveEntity)
	return o, err
}

func (st *Storage) ListCharacterContactIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterContactIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("ListCharacterContactIDs for character %d: %w", characterID, err)
	}
	return set.Collect(slices.Values(ids)), nil
}

func (st *Storage) ListCharacterContacts(ctx context.Context, characterID int64) ([]*app.CharacterContact, error) {
	rows, err := st.qRO.ListCharacterContacts(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterContact for character %d: %w", characterID, err)
	}
	var oo []*app.CharacterContact
	for _, r := range rows {
		oo = append(oo, characterContactFromDBModel(r.CharacterContact, r.EveEntity))
	}
	return oo, nil
}

func characterContactFromDBModel(r queries.CharacterContact, c queries.EveEntity) *app.CharacterContact {
	o2 := &app.CharacterContact{
		CharacterID: r.CharacterID,
		Contact:     eveEntityFromDBModel(c),
		IsBlocked:   optional.FromNullBool(r.IsBlocked),
		IsWatched:   optional.FromNullBool(r.IsWatched),
		Standing:    r.Standing,
	}
	return o2
}

type UpdateOrCreateCharacterContactParams struct {
	CharacterID int64
	ContactID   int64
	IsBlocked   optional.Optional[bool]
	IsWatched   optional.Optional[bool]
	Standing    float64
}

func (st *Storage) UpdateOrCreateCharacterContact(ctx context.Context, arg UpdateOrCreateCharacterContactParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterContact: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.ContactID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterContact(ctx, queries.UpdateOrCreateCharacterContactParams{
		CharacterID: arg.CharacterID,
		ContactID:   arg.ContactID,
		IsBlocked:   optional.ToNullBool(arg.IsBlocked),
		IsWatched:   optional.ToNullBool(arg.IsWatched),
		Standing:    arg.Standing,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

type CreateCharacterContactLabelParams struct {
	CharacterID int64
	LabelID     int64
	Name        string
}

func (st *Storage) CreateCharacterContactLabel(ctx context.Context, arg CreateCharacterContactLabelParams) error {
	err := st.qRW.CreateCharacterContactLabel(ctx, queries.CreateCharacterContactLabelParams{
		CharacterID: arg.CharacterID,
		LabelID:     arg.LabelID,
		Name:        arg.Name,
	})
	if err != nil {
		return fmt.Errorf("CreateCharacterContactLabel: %v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetCharacterContactLabel(ctx context.Context, characterID, labelID int64) (string, error) {
	r, err := st.qRO.GetCharacterContactLabel(ctx, queries.GetCharacterContactLabelParams{
		CharacterID: characterID,
		LabelID:     labelID,
	})
	if err != nil {
		return "", fmt.Errorf("GetCharacterContactLabel: %d %d: %w", characterID, labelID, err)
	}
	return r.Name, nil
}

func (st *Storage) ListCharacterContactLabels(ctx context.Context, characterID int64) (set.Set[string], error) {
	rows, err := st.qRO.ListCharacterContactLabels(ctx, characterID)
	if err != nil {
		return set.Set[string]{}, fmt.Errorf("ListCharacterContactLabels for character %d: %w", characterID, err)
	}
	x := set.Collect(slices.Values(rows))
	return x, nil
}

func (st *Storage) DeleteCharacterContactLabels(ctx context.Context, characterID int64, names set.Set[string]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterContactLabels for character %d and contact IDs: %v: %w", characterID, names, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	if names.Size() == 0 {
		return nil
	}
	err := st.qRW.DeleteCharacterContactLabels(ctx, queries.DeleteCharacterContactLabelsParams{
		CharacterID: characterID,
		Names:       slices.Collect(names.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	slog.Info("Character labels deleted", "characterID", characterID, "labels", names)
	return nil
}
