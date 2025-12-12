package internal

type Computed struct {
	*Owner
	*Signal

	compute func(*Computed) any
}

func (r *Runtime) NewComputed(compute func(*Computed) any) *Computed {
	c := &Computed{
		Owner:  r.NewOwner(),
		Signal: r.NewSignal(nil),

		compute: compute,
	}
	c.fn = c.run

	r.recompute(c.ReactiveNode)

	return c
}

func (c *Computed) run() {
	value := c.compute(c)
	c.pendingValue = &value
}

// todo: can this live here?
// func (c *Computed) recompute() { }
