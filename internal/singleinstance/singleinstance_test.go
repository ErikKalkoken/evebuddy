package singleinstance_test

import (
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
	"github.com/stretchr/testify/assert"
)

func TestGroup(t *testing.T) {
	var c atomic.Int64
	g := singleinstance.NewGroup()
	f := func() {
		x, err, aborted := g.Do("alpha", func() (any, error) {
			c.Add(1)
			time.Sleep(100 * time.Millisecond)
			return true, nil
		})
		log.Println(x, err, aborted)
	}
	wg := sync.WaitGroup{}
	wg.Go(f)
	wg.Go(f)
	wg.Wait()
	assert.EqualValues(t, 1, c.Load())
}
