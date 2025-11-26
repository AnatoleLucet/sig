package sigv2

import "sync"

// Committable is anything that can commit pending values
type Committable interface {
	commitPendingValue()
}

// Recomputable is anything that can be recomputed
type Recomputable interface {
	recompute(create bool)
}

// baseNode contains common fields for all reactive nodes
type baseNode[T any] struct {
	mu sync.RWMutex

	// Value state
	value        T
	pendingValue interface{} // *struct{} (NotPending) or T

	// Dependency graph
	deps     *link
	depsTail *link
	subs     *link
	subsTail *link

	// Heap membership
	nextHeap heapNode
	prevHeap heapNode

	// Tree structure
	parent        *ownerNode
	nextSibling   interface{} // Can be any baseNode type
	firstChild    interface{} // Can be any baseNode type

	// Disposal state
	disposal          interface{} // DisposalFunc or []DisposalFunc
	pendingDisposal   interface{}
	pendingFirstChild interface{} // Can be any baseNode type

	// Execution state
	height int
	time   uint64
	flags  ReactiveFlags

	// Async state
	statusFlags StatusFlags
	err         error
}

// getMutex implements Observable and Computable
func (n *baseNode[T]) getMutex() *sync.RWMutex {
	return &n.mu
}

// Observable interface implementations
func (n *baseNode[T]) getSubs() *link {
	return n.subs
}

func (n *baseNode[T]) getSubsTail() *link {
	return n.subsTail
}

func (n *baseNode[T]) setSubs(l *link) {
	n.subs = l
}

func (n *baseNode[T]) setSubsTail(l *link) {
	n.subsTail = l
}

func (n *baseNode[T]) getTime() uint64 {
	return n.time
}

func (n *baseNode[T]) getPendingValue() interface{} {
	return n.pendingValue
}

func (n *baseNode[T]) getStatusFlags() StatusFlags {
	return n.statusFlags
}

func (n *baseNode[T]) getError() error {
	return n.err
}

// Computable interface implementations
func (n *baseNode[T]) getDeps() *link {
	return n.deps
}

func (n *baseNode[T]) getDepsTail() *link {
	return n.depsTail
}

func (n *baseNode[T]) setDeps(l *link) {
	n.deps = l
}

func (n *baseNode[T]) setDepsTail(l *link) {
	n.depsTail = l
}

func (n *baseNode[T]) getHeight() int {
	return n.height
}

func (n *baseNode[T]) setHeight(h int) {
	n.height = h
}

func (n *baseNode[T]) getFlags() ReactiveFlags {
	return n.flags
}

func (n *baseNode[T]) setFlags(f ReactiveFlags) {
	n.flags = f
}

// heapNode interface implementations
func (n *baseNode[T]) getNextHeap() heapNode {
	return n.nextHeap
}

func (n *baseNode[T]) setNextHeap(h heapNode) {
	n.nextHeap = h
}

func (n *baseNode[T]) getPrevHeap() heapNode {
	return n.prevHeap
}

func (n *baseNode[T]) setPrevHeap(h heapNode) {
	n.prevHeap = h
}

// linkNodes creates a dependency link between a dependency and a subscriber
func linkNodes(dep Observable, sub Computable) {
	prevDep := sub.getDepsTail()

	// Already linked as most recent dependency
	if prevDep != nil && prevDep.dep == dep {
		return
	}

	var nextDep *link
	isRecomputing := sub.getFlags()&FlagRecomputingDeps != 0

	if isRecomputing {
		if prevDep != nil {
			nextDep = prevDep.nextDep
		} else {
			nextDep = sub.getDeps()
		}

		// Already linked, just update tail
		if nextDep != nil && nextDep.dep == dep {
			sub.setDepsTail(nextDep)
			return
		}
	}

	prevSub := dep.getSubsTail()

	// Already linked from subscriber side
	if prevSub != nil && prevSub.sub == sub {
		if !isRecomputing || isValidLink(prevSub, sub) {
			return
		}
	}

	// Create new link
	newLink := &link{
		dep:     dep,
		sub:     sub,
		nextDep: nextDep,
		prevSub: prevSub,
		nextSub: nil,
	}

	// Update subscriber's dependency list
	if prevDep != nil {
		prevDep.nextDep = newLink
	} else {
		sub.setDeps(newLink)
	}
	sub.setDepsTail(newLink)

	// Update dependency's subscriber list
	if prevSub != nil {
		prevSub.nextSub = newLink
	} else {
		dep.setSubs(newLink)
	}
	dep.setSubsTail(newLink)
}

// unlinkSubs removes a link and returns the next dependency
func unlinkSubs(l *link) *link {
	dep := l.dep
	nextDep := l.nextDep
	nextSub := l.nextSub
	prevSub := l.prevSub

	// Remove from subscriber list
	if nextSub != nil {
		nextSub.prevSub = prevSub
	} else {
		dep.setSubsTail(prevSub)
	}

	if prevSub != nil {
		prevSub.nextSub = nextSub
	} else {
		dep.setSubs(nextSub)
	}

	return nextDep
}

// isValidLink checks if a link is still in the subscriber's dependency list
func isValidLink(checkLink *link, sub Computable) bool {
	depsTail := sub.getDepsTail()
	if depsTail == nil {
		return false
	}

	l := sub.getDeps()
	for l != nil {
		if l == checkLink {
			return true
		}
		if l == depsTail {
			break
		}
		l = l.nextDep
	}

	return false
}

// markNode marks a node and its subscribers as needing updates
func markNode(el Computable, newState ReactiveFlags) {
	flags := el.getFlags()

	// Already marked with this state or dirtier
	if flags&(FlagCheck|FlagDirty) >= newState {
		return
	}

	el.setFlags((flags &^ (FlagCheck | FlagDirty)) | newState)

	// Get subscribers if this node is also observable
	if obs, ok := interface{}(el).(Observable); ok {
		for l := obs.getSubs(); l != nil; l = l.nextSub {
			markNode(l.sub, FlagCheck)
		}
	}
}
