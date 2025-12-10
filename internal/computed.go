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

	// this should be done in another way but wathever for now
	r.context.RunWithNode(c.ReactiveNode, c.fn)

	return c
}
