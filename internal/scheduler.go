package internal

import (
	"errors"
	"sync/atomic"
)

type Tick int64

type Scheduler struct {
	// incremented each time the scheduler is flushed (when reactive nodes are updated)
	// used for staleness detection
	clock atomic.Int64

	scheduled atomic.Bool
	running   atomic.Bool
}

func NewScheduler() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Schedule() {
	s.scheduled.Store(true)
}

func (s *Scheduler) IsScheduled() bool {
	return s.scheduled.Load()
}

func (s *Scheduler) IsRunning() bool {
	return s.running.Load()
}

func (s *Scheduler) Time() Tick {
	return Tick(s.clock.Load())
}

func (s *Scheduler) Run(fn func()) error {
	if !s.running.CompareAndSwap(false, true) {
		return nil
	}
	defer s.running.Store(false)

	count := 0
	for s.scheduled.Swap(false) {
		count++
		if count > 1e5 {
			return errors.New("possible infinite update loop detected")
		}

		s.clock.Add(1)

		fn()
	}

	return nil
}
