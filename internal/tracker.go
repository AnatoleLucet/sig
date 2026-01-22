package internal

import (
	"sync"
)

type Tracker struct {
	mu sync.RWMutex

	tracking bool

	executingGID       int64     // to prevent cross-goroutine tracking issues
	currentOwner       *Owner    // for lifecycle/cleanup tracking
	currentComputation *Computed // for reactive dependency tracking
}

func NewTracker() *Tracker {
	return &Tracker{
		tracking: true,
	}
}

func (t *Tracker) IsTracking() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tracking
}

func (t *Tracker) CurrentOwner() *Owner {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentOwner
}

func (t *Tracker) CurrentComputation() *Computed {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentComputation
}

func (t *Tracker) RunWithOwner(owner *Owner, fn func()) {
	defer owner.recover()

	t.mu.Lock()
	prev := t.currentOwner
	t.currentOwner = owner

	t.executingGID = getGID()
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.currentOwner = prev
		t.mu.Unlock()
	}()

	fn()
}

func (t *Tracker) RunWithComputation(node *Computed, fn func()) {
	defer node.recover()

	t.mu.Lock()
	prevOwner := t.currentOwner
	prevComputation := t.currentComputation

	t.currentOwner = node.Owner
	t.currentComputation = node

	t.executingGID = getGID()
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.currentOwner = prevOwner
		t.currentComputation = prevComputation
		t.mu.Unlock()
	}()

	fn()
}

func (t *Tracker) RunUntracked(fn func()) {
	t.mu.Lock()
	prev := t.tracking
	t.tracking = false
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.tracking = prev
		t.mu.Unlock()
	}()

	fn()
}

func (t *Tracker) Track(node *Signal) {
	t.mu.RLock()
	shouldTrack := t.shouldTrack(node)
	comp := t.currentComputation
	t.mu.RUnlock()

	if shouldTrack {
		comp.Link(comp, node)
	}
}

func (t *Tracker) shouldTrack(node *Signal) bool {
	callerGID := getGID()

	hasOwner := t.currentComputation != nil
	isTracking := t.tracking
	// make sure we're currently in the same goroutine as the computation
	// to avoid cross-goroutine tracking issues
	isSameGID := callerGID == t.executingGID

	return hasOwner && isTracking && isSameGID
}
