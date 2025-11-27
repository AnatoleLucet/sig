package sig

type ReactiveNode struct {
	value        any
	pendingValue *any

	height int

	parent *ReactiveNode

	// TODO: meh?
	nextSibling *ReactiveNode
	firstChild  *ReactiveNode
}

func createSignal(initial any) (func() any, func(any)) {
	node := ReactiveNode{
		value:  initial,
		parent: context,
	}

	get := func() any {
		return nil
	}

	set := func(v any) {
		node.pendingValue = &v
		dirtyHeap.Insert(&node)

		flush()
	}

	return get, set
}
