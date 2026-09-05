package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"
)

type gameTimeAdded struct {
	baseRenderer
}

func (n gameTimeAdded) render(_ context.Context, _ string, _ time.Time) (string, string, error) {
	title := "Game time added"
	body := "Game time has been added to your account."
	return title, body, nil
}

type gameTimeReceived struct {
	baseRenderer
}

func (n gameTimeReceived) unmarshal(text string) (goesi.GameTimeReceived, set.Set[int64], error) {
	var data goesi.GameTimeReceived
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.SenderCharID), nil
}

func (n gameTimeReceived) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n gameTimeReceived) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Game time received from %s", entities[data.SenderCharID].Name)
	body := fmt.Sprintf(
		"%s has sent you **%d** game time code(s).\n\n> %s",
		makeInfoLink(entities[data.SenderCharID]),
		data.Quantity,
		data.Message,
	)
	return title, body, nil
}

type gameTimeSent struct {
	baseRenderer
}

func (n gameTimeSent) unmarshal(text string) (goesi.GameTimeSent, set.Set[int64], error) {
	var data goesi.GameTimeSent
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, set.Set[int64]{}, err
	}
	return data, set.Of(data.ReceiverCharID, data.SenderCharID), nil
}

func (n gameTimeSent) entityIDs(text string) (set.Set[int64], error) {
	_, ids, err := n.unmarshal(text)
	return ids, err
}

func (n gameTimeSent) render(ctx context.Context, text string, _ time.Time) (string, string, error) {
	data, ids, err := n.unmarshal(text)
	if err != nil {
		return "", "", err
	}
	entities, err := n.eus.ToEntities(ctx, ids)
	if err != nil {
		return "", "", err
	}
	title := fmt.Sprintf("Game time sent to %s", entities[data.ReceiverCharID].Name)
	body := fmt.Sprintf(
		"%s has sent game time to %s.",
		makeInfoLink(entities[data.SenderCharID]),
		makeInfoLink(entities[data.ReceiverCharID]),
	)
	return title, body, nil
}
