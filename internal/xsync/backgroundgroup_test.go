package xsync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackgroundGroup_StopWithoutStart(t *testing.T) {
	var g BackgroundGroup
	// when
	done := make(chan struct{})
	go func() {
		g.Stop()
		close(done)
	}()
	// then
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return")
	}
}

func TestBackgroundGroup_StopIsIdempotent(t *testing.T) {
	var g BackgroundGroup
	g.StartTicker(10*time.Millisecond, false, func(ctx context.Context) {})
	g.Stop()
	// when/then
	assert.NotPanics(t, func() {
		g.Stop()
	})
}

func TestBackgroundGroup_NotImmediate_WaitsForFirstTick(t *testing.T) {
	var g BackgroundGroup
	var count atomic.Int32
	// when
	g.StartTicker(20*time.Millisecond, false, func(ctx context.Context) {
		count.Add(1)
	})
	defer g.Stop()
	// then
	assert.Equal(t, int32(0), count.Load(), "fire must not run before the first tick")
	time.Sleep(50 * time.Millisecond)
	assert.GreaterOrEqual(t, count.Load(), int32(1))
}

func TestBackgroundGroup_Immediate_FiresRightAway(t *testing.T) {
	var g BackgroundGroup
	entered := make(chan struct{}, 1)
	// when
	g.StartTicker(time.Hour, true, func(ctx context.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}
	})
	defer g.Stop()
	// then
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("fire was not called immediately")
	}
}

func TestBackgroundGroup_StopWaitsForInFlightFire(t *testing.T) {
	var g BackgroundGroup
	const delay = 200 * time.Millisecond
	entered := make(chan struct{}, 1)
	g.StartTicker(10*time.Millisecond, true, func(ctx context.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}
		time.Sleep(delay)
	})
	select {
	case <-entered:
		// fire is now sleeping, i.e. genuinely in flight
	case <-time.After(time.Second):
		t.Fatal("fire was never entered")
	}
	// when
	start := time.Now()
	g.Stop()
	// then
	assert.GreaterOrEqual(t, time.Since(start), delay)
}

func TestBackgroundGroup_StopCancelsFiresCtx(t *testing.T) {
	var g BackgroundGroup
	entered := make(chan struct{})
	proceed := make(chan struct{})
	ctxErrCh := make(chan error, 1)
	g.StartTicker(time.Hour, true, func(ctx context.Context) {
		close(entered)
		<-proceed
		ctxErrCh <- ctx.Err()
	})
	<-entered
	stopDone := make(chan struct{})
	go func() {
		g.Stop()
		close(stopDone)
	}()
	time.Sleep(20 * time.Millisecond) // let Stop's cancel() run before fire is allowed to proceed
	close(proceed)
	// when
	<-stopDone
	// then
	assert.ErrorIs(t, <-ctxErrCh, context.Canceled)
}

func TestBackgroundGroup_DoubleStartTicker_DoesNotOrphanFirst(t *testing.T) {
	var g BackgroundGroup
	var count atomic.Int32
	fire := func(ctx context.Context) { count.Add(1) }
	// when
	g.StartTicker(10*time.Millisecond, true, fire)
	g.StartTicker(10*time.Millisecond, true, fire)
	time.Sleep(30 * time.Millisecond)
	g.Stop()
	// then
	assert.GreaterOrEqual(t, count.Load(), int32(2), "both tickers should have fired")
}

func TestBackgroundGroup_Go_IsTrackedByStop(t *testing.T) {
	var g BackgroundGroup
	const delay = 100 * time.Millisecond
	entered := make(chan struct{})
	g.Go(func() {
		close(entered)
		time.Sleep(delay)
	})
	<-entered
	// when
	start := time.Now()
	g.Stop()
	// then
	assert.GreaterOrEqual(t, time.Since(start), delay)
}
