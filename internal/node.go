package internal

import "iter"

type NodeFlags int

const (
	FlagNone   NodeFlags = 0
	FlagInHeap NodeFlags = 1 << 0 // node is currently in heap
)

type ReactiveNode struct {
	// called whenever the node is dirty
	fn func()

	// the current height of the node in the dependency graph
	height int

	// the node's state
	flags NodeFlags

	subsHead *DependencyLink
	depsHead *DependencyLink

	parent       *ReactiveNode
	prevSibling  *ReactiveNode
	nextSibling  *ReactiveNode
	childrenHead *ReactiveNode
}

type DependencyLink struct {
	dep *ReactiveNode
	sub *ReactiveNode

	prevDep *DependencyLink
	nextDep *DependencyLink

	prevSub *DependencyLink
	nextSub *DependencyLink
}

func (r *Runtime) NewNode() *ReactiveNode {
	return &ReactiveNode{}
}

// HasFlag checks if the given flag is set
func (n *ReactiveNode) HasFlag(flag NodeFlags) bool {
	return n.flags&flag != 0
}

// AddFlag adds the given flag
func (n *ReactiveNode) AddFlag(flag NodeFlags) {
	n.flags |= flag
}

// RemoveFlag removes the given flag
func (n *ReactiveNode) RemoveFlag(flag NodeFlags) {
	n.flags &^= flag
}

// SetFlags sets the flags to exact value
func (n *ReactiveNode) SetFlags(flags NodeFlags) {
	n.flags = flags
}

// Link creates a bidirectional dependency link between this node (subcriber) and the given node (dependency).
func (sub *ReactiveNode) Link(dep *ReactiveNode) {
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
	if dep.fn != nil && dep.height >= sub.height {
		sub.height = dep.height + 1
	}
}

func (n *ReactiveNode) addDepLink(link *DependencyLink) {
	if n.depsHead == nil {
		n.depsHead = link
		link.prevDep = link // loop to self
		link.nextDep = nil
	} else {
		tail := n.depsHead.prevDep
		tail.nextDep = link
		link.prevDep = tail
		link.nextDep = nil
		n.depsHead.prevDep = link
	}
}

func (n *ReactiveNode) addSubLink(link *DependencyLink) {
	if n.subsHead == nil {
		n.subsHead = link
		link.prevSub = link // loop to self
		link.nextSub = nil
	} else {
		tail := n.subsHead.prevSub
		tail.nextSub = link
		link.prevSub = tail
		link.nextSub = nil
		n.subsHead.prevSub = link
	}
}

func (dep *ReactiveNode) removeSubLink(link *DependencyLink) {
	// single node
	if link.prevSub == link {
		dep.subsHead = nil
		link.prevSub = nil
		link.nextSub = nil
		return
	}

	// multiple nodes
	if link == dep.subsHead {
		dep.subsHead = link.nextSub
	} else {
		link.prevSub.nextSub = link.nextSub
	}

	if link.nextSub != nil {
		link.nextSub.prevSub = link.prevSub
	} else {
		dep.subsHead.prevSub = link.prevSub
	}

	link.prevSub = nil
	link.nextSub = nil
}

// Deps returns an iterator over all dependencies
func (sub *ReactiveNode) Deps() iter.Seq[*ReactiveNode] {
	return func(yield func(*ReactiveNode) bool) {
		link := sub.depsHead
		for link != nil {
			if !yield(link.dep) {
				return
			}

			link = link.nextDep
		}
	}
}

// Subs returns an iterator over all subscribers
func (dep *ReactiveNode) Subs() iter.Seq[*ReactiveNode] {
	return func(yield func(*ReactiveNode) bool) {
		link := dep.subsHead
		for link != nil {
			if !yield(link.sub) {
				return
			}

			link = link.nextSub
		}
	}
}

// ClearDeps removes all dependencies
func (sub *ReactiveNode) ClearDeps() {
	for link := sub.depsHead; link != nil; {
		next := link.nextDep
		link.dep.removeSubLink(link)
		link = next
	}

	sub.depsHead = nil
}

// MaxDepHeight returns the maximum height of the node's dependencies
func (sub *ReactiveNode) MaxDepHeight() int {
	maxHeight := 0
	for dep := range sub.Deps() {
		if dep.fn != nil && dep.height >= maxHeight {
			maxHeight = dep.height + 1
		}
	}

	return maxHeight
}
