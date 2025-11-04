package sig

import (
	"slices"
	"sync"
)

type Disposable interface {
	Dispose()
}

type owner struct {
	mu sync.Mutex

	parent   *owner
	children []Disposable

	ctx *reactiveContext
}

func (o *owner) addChild(child Disposable) {
	if !slices.Contains(o.children, child) {
		o.children = append(o.children, child)
	}
}

func (o *owner) removeChild(child Disposable) {
	i := slices.Index(o.children, child)
	if i >= 0 {
		o.children = slices.Delete(o.children, i, i+1)
	}
}

func (o *owner) Dispose() {
	if o.parent != nil {
		o.parent.removeChild(o)
	}

	for _, child := range o.children {
		child.Dispose()
	}
	o.children = nil
}

func (o *owner) Run(fn func()) {
	prevOwner := getActiveOwner()
	setActiveOwner(o)
	fn()
	setActiveOwner(prevOwner)
}

func Owner() *owner {
	o := &owner{
		parent: getActiveOwner(),
		ctx:    &reactiveContext{},
	}

	if o.parent != nil {
		o.parent.addChild(o)
	}

	return o
}
