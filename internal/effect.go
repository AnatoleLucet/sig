package internal

type EffectType int

const (
	EffectRender EffectType = iota
	EffectUser
)

type Effect struct {
	*Computed

	typ EffectType
}

func (r *Runtime) NewEffect(typ EffectType, effect func()) *Effect {
	var e *Effect

	e = &Effect{
		Computed: r.NewComputed(func(node *Computed) any {
			effect()
			return nil
		}),

		typ: typ,
	}

	e.mu.Lock()
	e.fn = e.run
	e.mu.Unlock()

	return e
}

func (e *Effect) run() {
	r := GetRuntime()

	r.effectQueue.Enqueue(e.Type(), func() {
		if e.HasFlag(FlagDisposed) {
			return
		}

		r.tracker.RunWithComputation(e.Computed, e.Computed.run)
	})

	r.Schedule(false)
}

func (e *Effect) Type() EffectType {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.typ
}
