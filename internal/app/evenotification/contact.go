package evenotification

import (
	"context"
	"fmt"
	"time"

	"github.com/fnt-eve/goesi-openapi"
	"github.com/goccy/go-yaml"
)

type contactAdd struct {
	baseRenderer
}

func (n contactAdd) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.ContactAdd
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Contact added"
	body := fmt.Sprintf(
		"You have been added as a contact with a standing of **%d**.\n\n%s",
		data.Level,
		data.Message,
	)
	return title, body, nil
}

type contactEdit struct {
	baseRenderer
}

func (n contactEdit) render(_ context.Context, text string, _ time.Time) (string, string, error) {
	var data goesi.ContactEdit
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return "", "", err
	}
	title := "Contact standing updated"
	body := fmt.Sprintf(
		"Your standing with a contact has been changed to **%.0f**.\n\n%s",
		data.Level,
		data.Message,
	)
	return title, body, nil
}
