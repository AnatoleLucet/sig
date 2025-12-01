package sig

func (r *Runtime) recompute(node *ReactiveNode) {
	oldHeight := node.height
	// oldValue := node.value

	node.ClearDeps()

	r.execContext.withNode(node, func() {
		if node.fn != nil {
			node.value = node.fn(node.Value())
		}
	})

	// newValue := node.value
	newHeight := node.MaxDepHeight()

	// valueChanged := newValue != oldValue
	heightChanged := newHeight != oldHeight

	// if valueChanged || heightChanged {
	if heightChanged {
		node.height = newHeight

		for sub := range node.Subs() {
			r.dirtyHeap.Insert(sub)
		}
	}
}

func (r *Runtime) Effect(fn func()) {
	node := NewNode()
	node.fn = func(any) any {
		fn()
		return nil
	}

	r.recompute(node)
}
