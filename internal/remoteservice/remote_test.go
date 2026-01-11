package remoteservice_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/remoteservice"
)

func TestRemoteService(t *testing.T) {
	const port = 30125
	var isCalled bool
	stop, err := remoteservice.Start(port, func() {
		isCalled = true
	})
	defer stop()
	if assert.NoError(t, err) {
		err := remoteservice.ShowPrimaryInstance(port)
		if assert.NoError(t, err) {
			assert.True(t, isCalled)
		}
	}
}
