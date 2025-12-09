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

func (r *Runtime) Effect(fn func() func()) {
	node := NewNode()
	node.fn = func(any) any {
		if node.HasFlag(FlagEnqueued) {
			return nil
		}
		node.AddFlag(FlagEnqueued)

		r.queue.Enqueue(EffectUser, func() {
			node.RemoveFlag(FlagEnqueued)

			node.disposal.RunEffect()

			cleanup := fn()
			if cleanup != nil {
				node.disposal.SetEffectCleanup(cleanup)
			}
		})

		r.scheduler.MarkScheduled()

		return nil
	}

	// TODO: add self to parent

	r.recompute(node)
}
