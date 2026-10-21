package xsync_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xsync"
)

func TestLatestRun(t *testing.T) {
	t.Run("should report a single run as newest", func(t *testing.T) {
		// given
		var r xsync.LatestRun
		// when
		isLatest := r.Start()
		// then
		assert.True(t, isLatest())
	})
	t.Run("should report an older run as outdated once a new run starts", func(t *testing.T) {
		// given
		var r xsync.LatestRun
		isLatest1 := r.Start()
		// when
		isLatest2 := r.Start()
		// then
		assert.False(t, isLatest1())
		assert.True(t, isLatest2())
	})
	t.Run("should report a run as outdated after several new runs", func(t *testing.T) {
		// given
		var r xsync.LatestRun
		isLatest := r.Start()
		r.Start()
		// when
		r.Start()
		// then
		assert.False(t, isLatest())
	})
	t.Run("should report a started run as outdated after Discard", func(t *testing.T) {
		// given
		var r xsync.LatestRun
		isLatest := r.Start()
		// when
		r.Discard()
		// then
		assert.False(t, isLatest())
	})
}
