package internal

import "iter"

type Computed struct {
	*Owner
	*Signal

	initialized bool

	// called whenever the nodes has to recompute its value
	fn func()

	depsHead *DependencyLink

	compute func(*Computed) any
}

func (r *Runtime) NewComputed(compute func(*Computed) any) *Computed {
	c := &Computed{
		Owner:  r.NewOwner(),
		Signal: r.NewSignal(nil),

		compute: compute,
	}
	c.fn = c.run

	c.OnDispose(func() {
		if c.depsHead != nil {
			r.heap.Remove(c)
			c.ClearDeps()
			c.SetFlags(FlagNone)
		}
	})

	r.recompute(c)

	return c
}

func (c *Computed) run() {
	if c.initialized {
		c.Dispose()
	}
	c.initialized = true

	value := c.compute(c)
	c.pendingValue = &value
}

// Link creates a bidirectional dependency link between this node (subcriber) and the given node (dependency).
func (c *Computed) Link(sub *Computed, dep *Signal) {
	// dont link if already present as the most recent dependency
	if sub.depsHead != nil {
		tail := sub.depsHead.prevDep
		if tail.dep == dep {
			return
		}
	}

	// todo: also check here if we're recomputing and avoid relinking (check solid's implem for this optimisation)

	link := &DependencyLink{dep: dep, sub: sub}

	sub.addDepLink(link)
	dep.addSubLink(link)

	// Update subscriber height if needed
	if dep.height >= sub.height {
		sub.height = dep.height + 1
	}
}

// Deps returns an iterator over all dependencies
func (c *Computed) Deps() iter.Seq[*Signal] {
	return func(yield func(*Signal) bool) {
		link := c.depsHead
		for link != nil {
			if !yield(link.dep) {
				return
			}

			link = link.nextDep
		}
	}
}

// ClearDeps removes all dependencies
func (c *Computed) ClearDeps() {
	for link := c.depsHead; link != nil; {
		next := link.nextDep
		link.dep.removeSubLink(link)
		link = next
	}

	c.depsHead = nil
}

// MaxDepHeight returns the maximum height of the node's dependencies
func (c *Computed) MaxDepHeight() int {
	maxHeight := 0
	for dep := range c.Deps() {
		if dep.height >= maxHeight {
			maxHeight = dep.height + 1
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
