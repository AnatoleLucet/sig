package internal

import (
	"iter"
)

type Owner struct {
	// cleanup functions to be called when the node is disposed
	cleanups []func()

	// panic error handlers
	catchers []func(any)

	// the context values of this owner
	context map[any]any

	parent       *Owner
	prevSibling  *Owner
	nextSibling  *Owner
	childrenHead *Owner
}

func (r *Runtime) NewOwner() *Owner {
	return &Owner{
		cleanups: make([]func(), 0),
		context:  make(map[any]any),
	}
}

func (o *Owner) Run(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if len(o.catchers) == 0 {
				panic(r)
			}

			for _, catcher := range o.catchers {
				catcher(r)
			}
		}
	}()

	r := GetRuntime()
	r.tracker.RunWithOwner(o, fn)
}

func (parent *Owner) AddChild(child *Owner) {
	child.parent = parent
	child.prevSibling = nil
	child.nextSibling = parent.childrenHead

	if parent.childrenHead != nil {
		parent.childrenHead.prevSibling = child
	}

	parent.childrenHead = child
}

func (n *Owner) Children() iter.Seq[*Owner] {
	return func(yield func(*Owner) bool) {
		child := n.childrenHead

		for child != nil {
			if !yield(child) {
				return
			}

			child = child.nextSibling
		}
	}
}

func (n *Owner) Dispose() {
	n.DisposeChildren()

	for i := 0; i < len(n.cleanups); i++ {
		n.cleanups[i]()
	}
	n.cleanups = nil
}

func (n *Owner) DisposeChildren() {
	for child := range n.Children() {
		child.Dispose()
	}
	n.childrenHead = nil
}

func (n *Owner) OnCleanup(fn func()) {
	n.cleanups = append(n.cleanups, fn)
}

func (n *Owner) OnError(fn func(any)) {
	n.catchers = append(n.catchers, fn)
}
