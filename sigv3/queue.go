package sig

type Queue struct {
	parent *Queue

	queues   []func()
	children []*Queue

	pendingNodes []ReactiveNode
}
