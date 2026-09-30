package xsync

import (
	"sync"
)

// Stack represents a basic stack which can be used concurrently.
// Its zero value is ready to use. Methods on a nil Stack are no-ops.
type Stack[T any] struct {
	mu sync.Mutex
	s  []T
}

// Push adds an item on the stack.
func (st *Stack[T]) Push(v T) {
	if st == nil {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s = append(st.s, v)
}

// Pop tries to return the item from the top of the stack
// and reports whether an item was returned.
func (st *Stack[T]) Pop() (T, bool) {
	var v T
	if st == nil {
		return v, false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.s) == 0 {
		return v, false
	}
	idx := len(st.s) - 1
	v = st.s[idx]
	clear(st.s[idx:]) // zeros out the element for GC
	st.s = st.s[:idx]
	return v, true
}

// Size returns the number of items in the stack.
func (st *Stack[T]) Size() int {
	if st == nil {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.s)
}
