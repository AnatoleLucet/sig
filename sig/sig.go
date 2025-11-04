package sig

import (
	"sync"

	"github.com/petermattis/goid"
)

// Reaction represents a reactive computation that depends on observables (signals).
// Think of it as an effect or a computed value that needs to be re-evaluated when its dependencies change.
type Reaction interface {
	// Execute runs the reaction's logic.
	Execute()

	// Dispose cleans up the reaction, removing all dependencies and stopping further executions.
	Dispose()

	// AddDependency registers an observable (signal) as a dependency of this reaction,
	addDependency(o Observable)

	// RemoveDependency removes an observable (signal) from this reaction's dependencies.
	removeDependency(o Observable)
}

// Observable represents a data source that can be observed by reactions.
// Think of it as a signal or state that can notify reactions (effects) when it changes.
type Observable interface {
	// Track registers a reaction to be notified when this observable changes.
	track(r Reaction)

	// Untrack removes a reaction from the notification list of this observable.
	untrack(r Reaction)
}

var activeOwners sync.Map

func getActiveOwner() *owner {
	gid := goid.Get()
	if o, ok := activeOwners.Load(gid); ok {
		return o.(*owner)
	}

	o := &owner{ctx: &reactiveContext{}}
	setActiveOwner(o)
	return o
}

func setActiveOwner(o *owner) {
	gid := goid.Get()
	activeOwners.Store(gid, o)
}
