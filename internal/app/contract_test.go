package app_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/stretchr/testify/assert"
)

func TestAssigneeName(t *testing.T) {
	x1 := &app.CharacterContract{
		Assignee: &app.EveEntity{Name: "name"},
	}
	assert.Equal(t, "name", x1.AssigneeName())
	x2 := &app.CharacterContract{}
	assert.Equal(t, "", x2.AssigneeName())
}

func TestAcceptorDisplay(t *testing.T) {
	x1 := &app.CharacterContract{
		Acceptor: &app.EveEntity{Name: "name"},
	}
	assert.Equal(t, "name", x1.AcceptorDisplay())
	x2 := &app.CharacterContract{}
	assert.Equal(t, "(None)", x2.AcceptorDisplay())
}
