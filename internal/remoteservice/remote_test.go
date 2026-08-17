package remoteservice_test

import (
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/remoteservice"
)

func TestRemoteService(t *testing.T) {
	t.Run("successful RPC call triggers callback", func(t *testing.T) {
		const port = 30125
		var mu sync.Mutex
		isCalled := false

		stop, err := remoteservice.Start(port, func() {
			mu.Lock()
			isCalled = true
			mu.Unlock()
		})
		require.NoError(t, err)
		defer stop()

		err = remoteservice.ShowPrimaryInstance(port)
		assert.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()
		assert.True(t, isCalled)
	})

	t.Run("panics when showInstance callback is nil", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = remoteservice.Start(30126, nil)
		})
	})

	t.Run("fails to start on an invalid port number", func(t *testing.T) {
		stop, err := remoteservice.Start(-1, func() {})
		if stop != nil {
			defer stop()
		}
		assert.Error(t, err)
	})

	t.Run("fails to start when port is already bound", func(t *testing.T) {
		const port = 30127
		stop1, err := remoteservice.Start(port, func() {})
		require.NoError(t, err)
		defer stop1()

		stop2, err := remoteservice.Start(port, func() {})
		if stop2 != nil {
			defer stop2()
		}
		assert.Error(t, err)
	})

	t.Run("ShowPrimaryInstance returns error when server is not running", func(t *testing.T) {
		const port = 30128
		err := remoteservice.ShowPrimaryInstance(port)
		assert.Error(t, err)
	})

	t.Run("ShowPrimaryInstance fails after server is stopped", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const port = 30129
			stop, err := remoteservice.Start(port, func() {})
			require.NoError(t, err)

			stop()
			synctest.Wait() // Waits for the background accept loop goroutine to exit on listener close

			err = remoteservice.ShowPrimaryInstance(port)
			assert.Error(t, err)
		})
	})
}
