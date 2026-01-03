package internal

type Tracker struct {
	tracking bool

	currentOwner       *Owner    // for lifecycle/cleanup tracking
	currentComputation *Computed // for reactive dependency tracking
}

func NewTracker() *Tracker {
	return &Tracker{
		tracking: true,
	}
}

func (t *Tracker) RunWithOwner(owner *Owner, fn func()) {
	defer owner.recover()

	prev := t.currentOwner
	t.currentOwner = owner
	defer func() { t.currentOwner = prev }()

	fn()
}

func (t *Tracker) RunWithComputation(node *Computed, fn func()) {
	defer node.recover()

	prevOwner := t.currentOwner
	prevComputation := t.currentComputation

	t.currentOwner = node.Owner
	t.currentComputation = node

	defer func() {
		t.currentOwner = prevOwner
		t.currentComputation = prevComputation
	}()

	fn()
}

func (t *Tracker) RunUntracked(fn func()) {
	prev := t.tracking
	t.tracking = false
	defer func() { t.tracking = prev }()

	fn()
}

func (t *Tracker) Track(node *Signal) {
	if t.ShouldTrack() {
		t.currentComputation.Link(t.currentComputation, node)
	}
}

func (ctx *Tracker) ShouldTrack() bool {
	return ctx.currentComputation != nil && ctx.tracking
}
