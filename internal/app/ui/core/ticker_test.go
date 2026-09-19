package core

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCancelableTicker_StopWithoutStart(t *testing.T) {
	var ct cancelableTicker
	// when
	done := make(chan struct{})
	go func() {
		ct.Stop()
		close(done)
	}()
	// then
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return")
	}
}

func TestCancelableTicker_StopIsIdempotent(t *testing.T) {
	var ct cancelableTicker
	ct.Start(10*time.Millisecond, false, func(ctx context.Context) {})
	ct.Stop()
	// when/then
	assert.NotPanics(t, func() {
		ct.Stop()
	})
}

func TestCancelableTicker_NotImmediate_WaitsForFirstTick(t *testing.T) {
	var ct cancelableTicker
	var count atomic.Int32
	// when
	ct.Start(20*time.Millisecond, false, func(ctx context.Context) {
		count.Add(1)
	})
	defer ct.Stop()
	// then
	assert.Equal(t, int32(0), count.Load(), "fire must not run before the first tick")
	time.Sleep(50 * time.Millisecond)
	assert.GreaterOrEqual(t, count.Load(), int32(1))
}

func TestCancelableTicker_Immediate_FiresRightAway(t *testing.T) {
	var ct cancelableTicker
	entered := make(chan struct{}, 1)
	// when
	ct.Start(time.Hour, true, func(ctx context.Context) {
		select {
		case entered <- struct{}{}:
		default:
		}
	})
	defer ct.Stop()
	// then
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("fire was not called immediately")
	}
}

func TestCancelableTicker_StopWaitsForInFlightFire(t *testing.T) {
	var ct cancelableTicker
	const delay = 200 * time.Millisecond
	entered := make(chan struct{}, 1)
	ct.Start(10*time.Millisecond, true, func(ctx context.Context) {
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
	ct.Stop()
	// then
	assert.GreaterOrEqual(t, time.Since(start), delay)
}

func TestCancelableTicker_StopCancelsFiresCtx(t *testing.T) {
	var ct cancelableTicker
	entered := make(chan struct{})
	proceed := make(chan struct{})
	ctxErrCh := make(chan error, 1)
	ct.Start(time.Hour, true, func(ctx context.Context) {
		close(entered)
		<-proceed
		ctxErrCh <- ctx.Err()
	})
	<-entered
	stopDone := make(chan struct{})
	go func() {
		ct.Stop()
		close(stopDone)
	}()
	time.Sleep(20 * time.Millisecond) // let Stop's cancel() run before fire is allowed to proceed
	close(proceed)
	// when
	<-stopDone
	// then
	assert.ErrorIs(t, <-ctxErrCh, context.Canceled)
}
