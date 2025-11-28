package sig

func (r *Runtime) recompute(node *ReactiveNode) {
	height := node.height
	node.ClearDeps()

	r.execContext.withNode(node, func() {
		if node.fn != nil {
			node.value = node.fn(node.Value())
		}
	})

	r.updateNodeHeight(node, height)
}

func (r *Runtime) updateNodeHeight(node *ReactiveNode, oldHeight int) {
	newHeight := node.MaxDepHeight()
	if newHeight != oldHeight {
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
