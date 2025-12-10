package internal

type Tracker struct {
	tracking bool

	currentNode *ReactiveNode
}

func NewTracker() *Tracker {
	return &Tracker{
		tracking: true,
	}
}

func (ctx *Tracker) RunWithNode(node *ReactiveNode, fn func()) {
	prev := ctx.currentNode
	ctx.currentNode = node
	defer func() { ctx.currentNode = prev }()

	fn()
}

func (ctx *Tracker) RunUntracked(fn func()) {
	prev := ctx.tracking
	ctx.tracking = false
	defer func() { ctx.tracking = prev }()

	fn()
}

func (ctx *Tracker) ShouldTrack() bool {
	return ctx.currentNode != nil && ctx.tracking
}
