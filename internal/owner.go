package internal

import (
	"iter"
)

type Owner struct {
	// cleanup functions to be called when the node is disposed
	cleanups []func()

	errorListeners   []func(any)
	disposeListeners []func()

	// the context values of this owner
	context map[any]any

	parent       *Owner
	prevSibling  *Owner
	nextSibling  *Owner
	childrenHead *Owner
}

func (r *Runtime) NewOwner() *Owner {
	o := &Owner{
		cleanups: make([]func(), 0),
		context:  make(map[any]any),
	}

	if parent := r.CurrentOwner(); parent != nil {
		parent.AddChild(o)
	}

	return o
}

func (o *Owner) Run(fn func() error) (err error) {
	r := GetRuntime()
	r.tracker.RunWithOwner(o, func() { err = fn() })

	return err
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

func (n *Owner) Cleanup() {
	n.DisposeChildren()

	for _, fn := range n.cleanups {
		fn()
	}
	n.cleanups = nil
}

func (n *Owner) Dispose() {
	n.DisposeChildren()

	for _, fn := range n.cleanups {
		fn()
	}
	n.cleanups = nil

	for _, fn := range n.disposeListeners {
		fn()
	}
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

func (n *Owner) OnDispose(fn func()) {
	n.disposeListeners = append(n.disposeListeners, fn)
}

func (n *Owner) OnError(fn func(any)) {
	n.errorListeners = append(n.errorListeners, fn)
}

func (n *Owner) recover() {
	r := recover()
	if r == nil {
		return
	}

	for owner := n; owner != nil; owner = owner.parent {
		if len(owner.errorListeners) > 0 {
			for _, fn := range owner.errorListeners {
				fn(r)
			}
			return
		}
	}

	panic(r)
}
