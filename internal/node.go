package internal

type NodeFlags int

// const (
// 	FlagNone
// 	FlagDirty
// 	FlagInHeap
// )

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

func (r *Runtime) NewNode() *ReactiveNode {
	return &ReactiveNode{}
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

type DependencyLink struct {
	dep *ReactiveNode
	sub *ReactiveNode

	prevDep *DependencyLink
	nextDep *DependencyLink

	prevSub *DependencyLink
	nextSub *DependencyLink
}
