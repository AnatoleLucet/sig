package internal

type Batcher struct {
	// each nested batch increases the depth by 1
	// if depth > 0, updates are queued until the outermost batch is complete
	depth int
}

func NewBatcher() *Batcher {
	return &Batcher{
		depth: 0,
	}
}

func (b *Batcher) IsBatching() bool {
	return b.depth > 0
}

func (b *Batcher) Batch(fn, onComplete func()) {
	b.depth++
	defer func() {
		b.depth--
		if b.depth == 0 && onComplete != nil {
			onComplete()
		}
	}()

	fn()
}

func (r *Runtime) NewBatch(fn func()) {
	r.batcher.Batch(fn, r.Flush)
}
