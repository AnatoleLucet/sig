package internal

type Computed struct {
	*Owner
	*Signal
}

func (r *Runtime) NewComputed(compute func() any) *Computed {
	c := &Computed{
		Owner:  r.NewOwner(),
		Signal: r.NewSignal(nil),
	}

	c.fn = func() {
		value := compute()
		c.pendingValue = &value
	}

	r.recompute(c.ReactiveNode)

	return c
}
