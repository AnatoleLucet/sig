package sig

import (
	"iter"
)

type NodeFlags int

const (
	FlagNone         NodeFlags = 0
	FlagCheck        NodeFlags = 1 << 0 // Node needs to check if deps changed
	FlagDirty        NodeFlags = 1 << 1 // Node needs recomputation
	FlagRecomputing  NodeFlags = 1 << 2 // Currently recomputing
	FlagInHeap       NodeFlags = 1 << 3 // In execution heap
	FlagInHeapHeight NodeFlags = 1 << 4 // In heap for height adjustement
	FlagZombie       NodeFlags = 1 << 5 // Maked for disposal
)

type ReactiveNode struct {
	value        any
	pendingValue *any

	flags NodeFlags

	height int

	fn func(any) any // for computed node

	depsHead *DependencyLink
	subsHead *DependencyLink
}

func NewNode() *ReactiveNode {
	n := &ReactiveNode{}
	return n
}

// Value returns the node's value (or pendingValue if set)
func (n *ReactiveNode) Value() any {
	if n.pendingValue != nil {
		return *n.pendingValue
	}

	return n.value
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

type DependencyLink struct {
	dep *ReactiveNode
	sub *ReactiveNode

	prevDep *DependencyLink
	nextDep *DependencyLink

	prevSub *DependencyLink
	nextSub *DependencyLink
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

	link := &DependencyLink{dep: dep, sub: sub}

	sub.addDepLink(link)
	dep.addSubLink(link)

	// Update subscriber height if needed
	if dep.fn != nil && dep.height >= sub.height {
		sub.height = dep.height + 1
	}
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

func (sub *ReactiveNode) addDepLink(link *DependencyLink) {
	if sub.depsHead == nil {
		sub.depsHead = link
		link.prevDep = link // loop to self
		link.nextDep = nil
	} else {
		tail := sub.depsHead.prevDep
		tail.nextDep = link
		link.prevDep = tail
		link.nextDep = nil
		sub.depsHead.prevDep = link
	}
}

func (sub *ReactiveNode) addSubLink(link *DependencyLink) {
	if sub.subsHead == nil {
		sub.subsHead = link
		link.prevSub = link // loop to self
		link.nextSub = nil
	} else {
		tail := sub.subsHead.prevSub
		tail.nextSub = link
		link.prevSub = tail
		link.nextSub = nil
		sub.subsHead.prevSub = link
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
