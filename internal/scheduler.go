package internal

type Tick int

type Scheduler struct {
	// incremented each time the scheduler is flushed (when reactive nodes are updated)
	// used for staleness detection
	clock Tick

	scheduled bool
	running   bool
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		clock: 0,

		scheduled: false,
		running:   false,
	}
}

func (s *Scheduler) Run(fn func()) {
	if s.running {
		return
	}

	s.running = true
	defer func() { s.running = false }()

	count := 0
	for s.scheduled {
		count++
		if count > 1e5 {
			panic("possible infinite update loop detected") // todo: handle this more gracefully
		}

		// s.running = false
		s.scheduled = false
		fn()
		s.clock++
	}
}

func (s *Scheduler) Schedule() {
	s.scheduled = true
}

func (s *Scheduler) Time() Tick {
	return s.clock
}
