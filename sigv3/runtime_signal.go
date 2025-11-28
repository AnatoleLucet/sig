package sig

func (r *Runtime) Signal(initial any) (func() any, func(any)) {
	node := NewNode()
	node.value = initial

	get := func() any {
		return r.read(node)
	}

	set := func(v any) {
		r.write(node, v)
	}

	return get, set
}

func (r *Runtime) read(node *ReactiveNode) any {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.execContext.ShouldTrack() {
		r.execContext.currentNode.Link(node)
	}

	return node.Value()
}

func (r *Runtime) write(node *ReactiveNode, v any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// TODO: equality check

	if node.pendingValue == nil {
		r.scheduler.Schedule(node)
	}

	node.pendingValue = &v

	r.markSubscribersDirty(node)
}
