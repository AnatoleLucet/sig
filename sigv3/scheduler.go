package sig

type NodeScheduler struct {
	clock     int
	scheduled bool
	running   bool

	pendingNodes []*ReactiveNode
}

func NewScheduler() *NodeScheduler {
	return &NodeScheduler{
		pendingNodes: make([]*ReactiveNode, 0),
	}
}

// Run executes the given function in the scheduler's context and gives a commit() function to apply pending values.
func (s *NodeScheduler) Run(fn func(commit func())) {
	if s.running || !s.scheduled {
		return
	}

	s.scheduled = false
	s.running = true

	fn(s.commit)

	s.clock++
	s.running = false
}

// Schedule takes a node with a pendingValue that can be comitted latter using the Run() method.
func (s *NodeScheduler) Schedule(node *ReactiveNode) {
	s.pendingNodes = append(s.pendingNodes, node)
	s.MarkScheduled()
}

func (s *NodeScheduler) MarkScheduled() {
	s.scheduled = true
}

func (s *NodeScheduler) commit() {
	for _, node := range s.pendingNodes {
		if node.pendingValue != nil {
			node.value = *node.pendingValue
			node.pendingValue = nil
		}
	}

	s.pendingNodes = s.pendingNodes[:0]
}
