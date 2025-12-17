package internal

func (r *Runtime) Untrack(fn func()) {
	r.tracker.RunUntracked(fn)
}
