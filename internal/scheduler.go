package internal

type Scheduler struct {
	// incremented each time the scheduler is flushed (when reactive nodes are updated)
	// used for staleness detection
	clock int

	// each nested batch increases the depth by 1
	// if depth > 0, updates are queued until the outermost batch is complete
	batchDepth int

	scheduled bool
	running   bool
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		clock:      0,
		batchDepth: 0,

		scheduled: false,
		running:   false,
	}
}

func (s *Scheduler) Run(fn func()) {
	if s.running || !s.scheduled {
		return
	}

	s.scheduled = false
	s.running = true

	fn()

	s.clock++
	s.running = false
}

func (s *Scheduler) Schedule() {
	s.scheduled = true

	// maybe move that to an upper schedule method on the runtime?
	if s.batchDepth == 0 {
		GetRuntime().Flush()
	}
}

func (s *Scheduler) Time() int {
	return s.clock
}
