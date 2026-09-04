package evenotification

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// implantNames returns the resolved names of a list of item type IDs, one per line.
func implantNames(entities map[int64]*app.EveEntity, ids []int64) string {
	var names []string
	for _, id := range ids {
		if e, ok := entities[id]; ok && e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return strings.Join(names, "\n")
}

type cloneActivationMsg struct {
	baseRenderer
}

func (n cloneActivationMsg) unmarshal(text string) (goesi.CloneActivationMsg, set.Set[int64], error) {
	var data goesi.CloneActivationMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CloneStationID, data.CorpStationID, data.PodKillerID), nil
}

func (n cloneActivationMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n cloneActivationMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	skill, err := n.eus.GetOrCreateTypeESI(ctx, data.SkillID)
	if err != nil {
		return "", "", err
	}
	title := "Clone activated"
	body := fmt.Sprintf(
		"Your pod was destroyed by %s and you have been reactivated in a clone at %s.\n\n"+
			"You lost **%d** skill points in %s.",
		makeInfoLink(entities[data.PodKillerID]),
		makeInfoLink(entities[data.CloneStationID]),
		data.SkillPointsLost,
		makeInfoLink(skill),
	)
	return title, body, nil
}

type cloneActivationMsg2 struct {
	baseRenderer
}

func (n cloneActivationMsg2) unmarshal(text string) (goesi.CloneActivationMsg2, set.Set[int64], error) {
	var data goesi.CloneActivationMsg2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CloneStationID, data.CorpStationID, data.PodKillerID), nil
}

func (n cloneActivationMsg2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n cloneActivationMsg2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Clone activated"
	body := fmt.Sprintf(
		"Your pod was destroyed by %s and you have been reactivated in a clone at %s.",
		makeInfoLink(entities[data.PodKillerID]),
		makeInfoLink(entities[data.CloneStationID]),
	)
	return title, body, nil
}

type cloneMovedMsg struct {
	baseRenderer
}

func (n cloneMovedMsg) unmarshal(text string) (goesi.CloneMovedMsg, set.Set[int64], error) {
	var data goesi.CloneMovedMsg
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.NewStationID, data.StationID), nil
}

func (n cloneMovedMsg) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n cloneMovedMsg) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := "Medical clone moved"
	body := fmt.Sprintf(
		"Your medical clone has been moved from %s to %s by %s.",
		makeInfoLink(entities[data.StationID]),
		makeInfoLink(entities[data.NewStationID]),
		makeInfoLink(entities[data.CorpID]),
	)
	return title, body, nil
}

type cloneRevokedMsg2 struct {
	baseRenderer
}

func (n cloneRevokedMsg2) unmarshal(text string) (goesi.CloneRevokedMsg2, set.Set[int64], error) {
	var data goesi.CloneRevokedMsg2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.CorpID, data.NewStationID), nil
}

func (n cloneRevokedMsg2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n cloneRevokedMsg2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	station, err := n.eus.GetOrCreateLocationESI(ctx, data.StationID)
	if err != nil {
		return "", "", err
	}
	title := "Clone contract revoked"
	body := fmt.Sprintf(
		"%s has revoked your medical clone contract at %s.\n\n"+
			"Your medical clone will now be moved to %s.",
		makeInfoLink(entities[data.CorpID]),
		makeInfoLink(station),
		makeInfoLink(entities[data.NewStationID]),
	)
	return title, body, nil
}

type jumpCloneDeletedMsg1 struct {
	baseRenderer
}

func (n jumpCloneDeletedMsg1) unmarshal(text string) (goesi.JumpCloneDeletedMsg1, set.Set[int64], error) {
	var data goesi.JumpCloneDeletedMsg1
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.LocationOwnerID, data.OwnerID)
	ids.AddSeq(slices.Values(data.TypeIDs))
	return data, ids, nil
}

func (n jumpCloneDeletedMsg1) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n jumpCloneDeletedMsg1) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	location, err := n.eus.GetOrCreateLocationESI(ctx, data.LocationID)
	if err != nil {
		return "", "", err
	}
	title := "Jump clone deleted"
	body := fmt.Sprintf(
		"A jump clone belonging to %s at %s (owned by %s) has been deleted.",
		makeInfoLink(entities[data.OwnerID]),
		makeInfoLink(location),
		makeInfoLink(entities[data.LocationOwnerID]),
	)
	if names := implantNames(entities, data.TypeIDs); names != "" {
		body += fmt.Sprintf("\n\nImplants lost:\n\n%s", names)
	}
	return title, body, nil
}

type jumpCloneDeletedMsg2 struct {
	baseRenderer
}

func (n jumpCloneDeletedMsg2) unmarshal(text string) (goesi.JumpCloneDeletedMsg2, set.Set[int64], error) {
	var data goesi.JumpCloneDeletedMsg2
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	ids := set.Of(data.DestroyerID, data.LocationOwnerID, data.OwnerID)
	ids.AddSeq(slices.Values(data.TypeIDs))
	return data, ids, nil
}

func (n jumpCloneDeletedMsg2) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n jumpCloneDeletedMsg2) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	location, err := n.eus.GetOrCreateLocationESI(ctx, data.LocationID)
	if err != nil {
		return "", "", err
	}
	title := "Jump clone deleted"
	body := fmt.Sprintf(
		"A jump clone belonging to %s at %s (owned by %s) has been destroyed by %s.",
		makeInfoLink(entities[data.OwnerID]),
		makeInfoLink(location),
		makeInfoLink(entities[data.LocationOwnerID]),
		makeInfoLink(entities[data.DestroyerID]),
	)
	if names := implantNames(entities, data.TypeIDs); names != "" {
		body += fmt.Sprintf("\n\nImplants lost:\n\n%s", names)
	}
	return title, body, nil
}
