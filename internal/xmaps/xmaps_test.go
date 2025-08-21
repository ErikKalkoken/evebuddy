package xmaps_test

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
	"github.com/stretchr/testify/assert"
)

func TestXMaps(t *testing.T) {
	m := xmaps.OrderedMap[string, int]{"bravo": 1, "alpha": 2}
	var keys []string
	var values []int
	for k, v := range m.All() {
		keys = append(keys, k)
		values = append(values, v)
	}
	assert.Equal(t, []string{"alpha", "bravo"}, keys)
	assert.Equal(t, []int{2, 1}, values)
}
