package remoteservice_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/remoteservice"
	"github.com/stretchr/testify/assert"
)

func TestRemoteService(t *testing.T) {
	var isCalled bool
	err := remoteservice.Start(func() {
		isCalled = true
	})
	if assert.NoError(t, err) {
		err := remoteservice.ShowPrimaryInstance()
		if assert.NoError(t, err) {
			assert.True(t, isCalled)
		}
	}
}
