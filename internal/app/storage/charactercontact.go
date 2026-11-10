package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
)

func (st *Storage) GetCharacterContact(ctx context.Context, characterID int64, contactID int64) (*app.CharacterContact, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterContact: %d %d: %w", characterID, contactID, err)
	}
	if characterID == 0 || contactID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	tx, err := st.dbRO.Begin()
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
	tx, err := st.dbRO.Begin()
	if err != nil {
		return nil, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRO.WithTx(tx)
	rows, err := qtx.ListCharacterContacts(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("ListCharacterContact for character %d: %w", characterID, err)
	}
	labelRows, err := qtx.ListCharacterContactLabelsForCharacter(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	labels := make(map[int64][]string)
	for _, r := range labelRows {
		labels[r.ContactID] = append(labels[r.ContactID], r.Name)
	}
	oo := make([]*app.CharacterContact, len(rows))
	for i, r := range rows {
		oo[i] = characterContactFromDBModel(r.CharacterContact, r.EveEntity, labels[r.CharacterContact.ID])
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

// ReplaceCharacterContactParams is a contact of a character from ESI.
type ReplaceCharacterContactParams struct {
	ContactID int64
	IsBlocked optional.Optional[bool]
	IsWatched optional.Optional[bool]
	LabelIDs  []int64
	Standing  float64
}

// ReplaceCharacterContacts replaces all stored contacts of a character with args
// and reports whether any contact was created, updated or deleted.
// Contact IDs in args must be unique.
// Labels not stored for the character are ignored.
func (st *Storage) ReplaceCharacterContacts(ctx context.Context, characterID int64, args []ReplaceCharacterContactParams) (bool, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterContacts: character %d: %w", characterID, err)
	}
	if characterID == 0 {
		return false, wrapErr(app.ErrInvalid)
	}
	for _, arg := range args {
		if arg.ContactID == 0 {
			return false, wrapErr(fmt.Errorf("contact with ID 0: %w", app.ErrInvalid))
		}
	}
	current, currentLabels, storedLabels, err := st.listCharacterContactStates(ctx, characterID)
	if err != nil {
		return false, wrapErr(err)
	}
	var changed []ReplaceCharacterContactParams
	for _, arg := range args {
		var labelIDs []int64
		for _, id := range arg.LabelIDs {
			if !storedLabels.Contains(id) {
				slog.Warn("Ignoring missing contact label", "characterID", characterID, "contactID", arg.ContactID, "labelID", id)
				continue
			}
			labelIDs = append(labelIDs, id)
		}
		arg.LabelIDs = labelIDs
		r, found := xmaps.Pop(current, arg.ContactID) // what remains in current is obsolete
		if found &&
			r.IsBlocked == optional.ToNullBool(arg.IsBlocked) &&
			r.IsWatched == optional.ToNullBool(arg.IsWatched) &&
			r.Standing == arg.Standing &&
			set.Of(arg.LabelIDs...).Equal(currentLabels[arg.ContactID]) {
			continue
		}
		changed = append(changed, arg)
	}
	obsolete := slices.Collect(maps.Keys(current))
	if len(changed) == 0 && len(obsolete) == 0 {
		return false, nil
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return false, wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	for ids := range slices.Chunk(obsolete, st.MaxIDsPerQuery) {
		err := qtx.DeleteCharacterContacts(ctx, queries.DeleteCharacterContactsParams{
			CharacterID: characterID,
			ContactIds:  ids,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("delete %d contacts: %w", len(ids), err))
		}
	}
	for _, arg := range changed {
		pk, err := qtx.UpdateOrCreateCharacterContact(ctx, queries.UpdateOrCreateCharacterContactParams{
			CharacterID: characterID,
			ContactID:   arg.ContactID,
			IsBlocked:   optional.ToNullBool(arg.IsBlocked),
			IsWatched:   optional.ToNullBool(arg.IsWatched),
			Standing:    arg.Standing,
		})
		if err != nil {
			return false, wrapErr(fmt.Errorf("update or create contact %d: %w", arg.ContactID, err))
		}
		if err := qtx.DeleteCharacterContactContactLabelsForContact(ctx, pk); err != nil {
			return false, wrapErr(fmt.Errorf("delete labels of contact %d: %w", arg.ContactID, err))
		}
		if err := addCharacterContactLabels(ctx, qtx, characterID, pk, arg.LabelIDs); err != nil {
			return false, wrapErr(fmt.Errorf("add labels to contact %d: %w", arg.ContactID, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return false, wrapErr(err)
	}
	slog.Info("Character contacts replaced", "characterID", characterID, "changed", len(changed), "deleted", len(obsolete))
	return true, nil
}

// addCharacterContactLabels adds labels to a contact. Must be called within a transaction.
func addCharacterContactLabels(ctx context.Context, q *queries.Queries, characterID, contactPK int64, labelIDs []int64) error {
	for id := range set.Of(labelIDs...).All() {
		label, err := q.GetCharacterContactLabel(ctx, queries.GetCharacterContactLabelParams{
			CharacterID: characterID,
			LabelID:     id,
		})
		if err != nil {
			return fmt.Errorf("get label %d: %w", id, err)
		}
		err = q.CreateCharacterContactContactLabel(ctx, queries.CreateCharacterContactContactLabelParams{
			ContactID: contactPK,
			LabelID:   label.ID,
		})
		if err != nil {
			return fmt.Errorf("add label %d: %w", id, err)
		}
	}
	return nil
}

// listCharacterContactStates returns the stored contacts of a character
// and their label IDs, both by contact ID, and the IDs of the character's labels.
func (st *Storage) listCharacterContactStates(ctx context.Context, characterID int64) (map[int64]queries.CharacterContact, map[int64]set.Set[int64], set.Set[int64], error) {
	tx, err := st.dbRO.Begin()
	if err != nil {
		return nil, nil, set.Set[int64]{}, err
	}
	defer tx.Rollback()
	qtx := st.qRO.WithTx(tx)
	rows, err := qtx.ListCharacterContactStates(ctx, characterID)
	if err != nil {
		return nil, nil, set.Set[int64]{}, err
	}
	labelIDs, err := qtx.ListCharacterContactLabelIDs(ctx, characterID)
	if err != nil {
		return nil, nil, set.Set[int64]{}, err
	}
	mapRows, err := qtx.ListCharacterContactContactLabelIdsForCharacter(ctx, characterID)
	if err != nil {
		return nil, nil, set.Set[int64]{}, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, set.Set[int64]{}, err
	}
	contacts := make(map[int64]queries.CharacterContact, len(rows))
	for _, r := range rows {
		contacts[r.ContactID] = r
	}
	contactLabels := make(map[int64]set.Set[int64])
	for _, r := range mapRows {
		x := contactLabels[r.ContactID]
		x.Add(r.LabelID)
		contactLabels[r.ContactID] = x
	}
	return contacts, contactLabels, set.Of(labelIDs...), nil
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
