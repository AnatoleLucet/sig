package internal

type ExecutionContext struct {
	tracking bool

	currentNode *ReactiveNode
}

func NewContext() *ExecutionContext {
	return &ExecutionContext{
		tracking: true,
	}
}

func (ctx *ExecutionContext) RunWithNode(node *ReactiveNode, fn func()) {
	prev := ctx.currentNode
	ctx.currentNode = node
	defer func() { ctx.currentNode = prev }()

	fn()
}

func (ctx *ExecutionContext) RunUntracked(fn func()) {
	prev := ctx.tracking
	ctx.tracking = false
	defer func() { ctx.tracking = prev }()

	fn()
}

func (ctx *ExecutionContext) ShouldTrack() bool {
	return ctx.currentNode != nil && ctx.tracking
}
