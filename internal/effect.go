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
	e.fn = e.run

	return e
}

func (e *Effect) run() {
	r := GetRuntime()

	r.effectQueue.Enqueue(e.typ, func() {
		if e.HasFlag(FlagDisposed) {
			return
		}

		r.tracker.RunWithComputation(e.Computed, e.Computed.run)
	})
	r.Schedule()
}
