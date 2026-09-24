package screens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatestRun(t *testing.T) {
	var l latestRun
	isLatest1 := l.start()
	assert.True(t, isLatest1())
	isLatest2 := l.start()
	assert.False(t, isLatest1())
	assert.True(t, isLatest2())
	l.start()
	assert.False(t, isLatest2())
}
