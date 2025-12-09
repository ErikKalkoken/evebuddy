// Package singleinstance provides a mechanism to ensure only one instance of a function runs at a time.
package singleinstance

import "sync"

// Group represents a class of work and forms a namespace in which units of work
// can be executed with duplicate suppression.
type Group struct {
	mu sync.Mutex
	m  map[string]struct{} // holds the keys for currently running functions
}

func NewGroup() *Group {
	g := &Group{m: make(map[string]struct{})}
	return g
}

// Do executes and returns the results of the given function,
// making sure that only one execution is in-flight for a given key at a time.
// If a duplicate comes in, the duplicate caller will be aborted.
// The return value aborted indicates whether v was aborted.
func (g *Group) Do(key string, fn func() (any, error)) (v any, err error, aborted bool) {
	g.mu.Lock()
	_, found := g.m[key]
	if found {
		g.mu.Unlock()
		return nil, nil, true
	}
	g.m[key] = struct{}{}
	g.mu.Unlock()
	x, err := fn()
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return x, err, false
}
