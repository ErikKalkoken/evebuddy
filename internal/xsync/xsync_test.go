package xsync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap_BasicOperations(t *testing.T) {
	var myMap Map[string, int]

	// Test Store and Load
	myMap.Store("apple", 10)
	val, ok := myMap.Load("apple")
	assert.True(t, ok)
	assert.Equal(t, 10, val)

	// Test Load for non-existent key
	val, ok = myMap.Load("banana")
	assert.False(t, ok)
	assert.Equal(t, 0, val) // Should return zero value of V

	// Test LoadOrStore
	actual, loaded := myMap.LoadOrStore("apple", 20)
	assert.True(t, loaded)
	assert.Equal(t, 10, actual, "Should return existing value")

	actual, loaded = myMap.LoadOrStore("cherry", 30)
	assert.False(t, loaded)
	assert.Equal(t, 30, actual, "Should return the newly stored value")

	// Test Delete
	myMap.Delete("apple")
	_, ok = myMap.Load("apple")
	assert.False(t, ok)
}

func TestMap_Range(t *testing.T) {
	var myMap Map[int, string]
	input := map[int]string{1: "one", 2: "two", 3: "three"}

	for k, v := range input {
		myMap.Store(k, v)
	}

	captured := make(map[int]string)
	myMap.Range(func(key int, value string) bool {
		captured[key] = value
		return true // continue iteration
	})

	assert.Equal(t, len(input), len(captured))
	assert.Equal(t, input, captured)
}

func TestMap_Concurrency(t *testing.T) {
	// Ensure the wrapper handles concurrent access safely (as sync.Map does)
	var myMap Map[int, int]
	var wg sync.WaitGroup
	iterations := 1000

	wg.Add(2)

	// Concurrent Writers
	go func() {
		defer wg.Done()
		for i := range iterations {
			myMap.Store(i, i*2)
		}
	}()

	// Concurrent Readers
	go func() {
		defer wg.Done()
		for i := range iterations {
			myMap.Load(i)
		}
	}()

	wg.Wait()
}
