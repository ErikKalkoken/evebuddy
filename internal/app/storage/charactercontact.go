package storage

import (
	"context"
	"database/sql"
	"errors"
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
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterContact: %d %d: %w", characterID, contactID, err)
	}
	if characterID == 0 || contactID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return nil, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRO.WithTx(tx)
	r, err := qtx.GetCharacterContact(ctx, queries.GetCharacterContactParams{
		CharacterID: characterID,
		ContactID:   contactID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	labels, err := qtx.ListCharacterContactContactLabels(ctx, r.CharacterContact.ID)
	if err != nil {
		return nil, wrapErr(err)
	}
	o := characterContactFromDBModel(r.CharacterContact, r.EveEntity, labels)
	if err := tx.Commit(); err != nil {
		return nil, wrapErr(err)
	}
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
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterContacts: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return nil, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRO.WithTx(tx)
	rows, err := qtx.ListCharacterContacts(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterContact for character %d: %w", characterID, err)
	}
	var oo []*app.CharacterContact
	for _, r := range rows {
		labels, err := qtx.ListCharacterContactContactLabels(ctx, r.CharacterContact.ID)
		if err != nil {
			return nil, wrapErr(err)
		}
		o := characterContactFromDBModel(r.CharacterContact, r.EveEntity, labels)
		oo = append(oo, o)
	}
	if err := tx.Commit(); err != nil {
		return nil, wrapErr(err)
	}
	return oo, nil
}

func characterContactFromDBModel(r queries.CharacterContact, c queries.EveEntity, labels []string) *app.CharacterContact {
	o2 := &app.CharacterContact{
		CharacterID: r.CharacterID,
		Contact:     eveEntityFromDBModel(c),
		IsBlocked:   optional.FromNullBool(r.IsBlocked),
		IsWatched:   optional.FromNullBool(r.IsWatched),
		Standing:    r.Standing,
		Labels:      set.Collect(slices.Values(labels)),
	}
	return o2
}

type UpdateOrCreateCharacterContactParams struct {
	CharacterID int64
	ContactID   int64
	IsBlocked   optional.Optional[bool]
	IsWatched   optional.Optional[bool]
	LabelIDs    []int64
	Standing    float64
}

func (st *Storage) UpdateOrCreateCharacterContact(ctx context.Context, arg UpdateOrCreateCharacterContactParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterContact: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.ContactID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	contactID, err := qtx.UpdateOrCreateCharacterContact(ctx, queries.UpdateOrCreateCharacterContactParams{
		CharacterID: arg.CharacterID,
		ContactID:   arg.ContactID,
		IsBlocked:   optional.ToNullBool(arg.IsBlocked),
		IsWatched:   optional.ToNullBool(arg.IsWatched),
		Standing:    arg.Standing,
	})
	if err != nil {
		return wrapErr(err)
	}

	x, err := qtx.ListCharacterContactContactLabelIds(ctx, queries.ListCharacterContactContactLabelIdsParams{
		CharacterID: arg.CharacterID,
		ContactID:   arg.ContactID,
	})
	if err != nil {
		return wrapErr(err)
	}

	current := set.Collect(slices.Values(x))
	incoming := set.Collect(slices.Values(arg.LabelIDs))

	toAdd := set.Difference(incoming, current)
	if toAdd.Size() > 0 {
		for labelID := range toAdd.All() {
			label, err := qtx.GetCharacterContactLabel(ctx, queries.GetCharacterContactLabelParams{
				CharacterID: arg.CharacterID,
				LabelID:     labelID,
			})
			if errors.Is(err, sql.ErrNoRows) {
				slog.Warn(
					"Ignoring missing contact label",
					slog.Int64("characterID", arg.CharacterID),
					slog.Int64("contactID", arg.ContactID),
					slog.Int64("labelID", labelID),
				)
				continue
			}
			if err != nil {
				return wrapErr(convertGetError(err))
			}
			err = qtx.CreateCharacterContactContactLabel(ctx, queries.CreateCharacterContactContactLabelParams{
				ContactID: contactID,
				LabelID:   label.ID,
			})
			if err != nil {
				return wrapErr(err)
			}
		}
	}

	toDelete := set.Difference(current, incoming)
	if toDelete.Size() > 0 {
		err := qtx.DeleteCharacterContactContactLabels(ctx, queries.DeleteCharacterContactContactLabelsParams{
			CharacterID: arg.CharacterID,
			ContactID:   arg.ContactID,
			LabelIds:    slices.Collect(toDelete.All()),
		})
		if err != nil {
			return wrapErr(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}
