package sig

import (
	"slices"
	"sync"
)

type reactiveContext struct {
	mu sync.Mutex

	// activeReaction holds the currently executing reaction.
	// It is used to track dependencies during reaction execution.
	activeReaction Reaction

	// pendingReactions holds reactions queued for execution during a batch.
	pendingReactions []Reaction

	// batchDepth indicates the current depth of nested Batch calls.
	// It is used to determine when to flush pending reactions.
	batchDepth int
}

func (rc *reactiveContext) batch(fn func()) {
	rc.batchDepth++
	fn()
	rc.batchDepth--

	if rc.batchDepth == 0 {
		reactions := rc.pendingReactions
		rc.pendingReactions = nil

		for _, reaction := range reactions {
			reaction.Execute()
		}
	}
}

func (rc *reactiveContext) queueReaction(r Reaction) {
	// if not in batch mode, execute immediately
	if rc.batchDepth == 0 {
		r.Execute()
		return
	}

	// else, queue for later execution
	if !slices.Contains(rc.pendingReactions, r) {
		rc.pendingReactions = append(rc.pendingReactions, r)
	}
}

func Batch(fn func()) {
	getActiveOwner().ctx.batch(fn)
}
