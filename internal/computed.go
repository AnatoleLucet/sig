package internal

import (
	"iter"
	"sync"
)

type Computed struct {
	*Owner
	*Signal

	mu          sync.RWMutex
	initialized bool

	// called whenever the nodes has to recompute its value
	fn func()

	depsHead *DependencyLink

	compute func(*Computed) any
}

func (r *Runtime) NewComputed(compute func(*Computed) any) *Computed {
	c := &Computed{
		Owner:   r.NewOwner(),
		Signal:  r.NewSignal(nil),
		compute: compute,
	}

	c.mu.Lock()
	c.fn = c.run
	c.mu.Unlock()

	c.OnDispose(func() {
		if c.depsHead != nil {
			r.heap.Remove(c)
			c.ClearDeps()
		}
		c.SetFlags(FlagDisposed)
	})

	r.recompute(c)

	return c
}

func (c *Computed) run() {
	c.mu.Lock()
	shouldCleanup := c.initialized
	c.initialized = true
	c.mu.Unlock()

	if shouldCleanup {
		c.Cleanup()
	}

	value := c.compute(c)

	c.Signal.mu.Lock()
	c.pendingValue = &value
	c.Signal.mu.Unlock()
}

// Link creates a bidirectional dependency link between this node (subcriber) and the given node (dependency).
func (c *Computed) Link(sub *Computed, dep *Signal) {
	sub.mu.Lock()
	// dont link if already present as the most recent dependency
	if sub.depsHead != nil {
		tail := sub.depsHead.prevDep
		if tail.dep == dep {
			sub.mu.Unlock()
			return
		}
	}

	// todo: also check here if we're recomputing and avoid relinking (check solid's implem for this optimisation)

	link := &DependencyLink{dep: dep, sub: sub}

	sub.addDepLink(link)
	sub.mu.Unlock()

	dep.mu.Lock()
	dep.addSubLink(link)
	dep.mu.Unlock()

	// update subscriber height if needed
	if dep.GetHeight() >= sub.GetHeight() {
		sub.SetHeight(dep.GetHeight() + 1)
	}
}

// Deps returns an iterator over all dependencies
func (c *Computed) Deps() iter.Seq[*Signal] {
	return func(yield func(*Signal) bool) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		link := c.depsHead
		for link != nil {
			if !yield(link.dep) {
				return
			}

			link = link.nextDep
		}
	}
}

func (c *Computed) getFn() func() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fn
}

// ClearDeps removes all dependencies
func (c *Computed) ClearDeps() {
	c.mu.Lock()
	// collect while locking
	var links []*DependencyLink
	for link := c.depsHead; link != nil; link = link.nextDep {
		links = append(links, link)
	}
	c.depsHead = nil
	c.mu.Unlock()

	// remove links without holding lock
	for _, link := range links {
		link.dep.removeSubLink(link)
	}
}

// MaxDepHeight returns the maximum height of the node's dependencies
func (c *Computed) MaxDepHeight() int {
	maxHeight := 0
	for dep := range c.Deps() {
		if dep.GetHeight() >= maxHeight {
			maxHeight = dep.GetHeight() + 1
		}
	}

	return maxHeight
}

func (c *Computed) addDepLink(link *DependencyLink) {
	if c.depsHead == nil {
		c.depsHead = link
		link.prevDep = link // loop to self
		link.nextDep = nil
	} else {
		tail := c.depsHead.prevDep
		tail.nextDep = link
		link.prevDep = tail
		link.nextDep = nil
		c.depsHead.prevDep = link
	}
}
