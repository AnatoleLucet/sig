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

func (r *Runtime) NewEffect(typ EffectType, effect func() func()) *Effect {
	var e *Effect

	e = &Effect{
		// an effect is just a computed that returns a cleanup function
		Computed: r.NewComputed(func(node *Computed) any {
			initialized := e != nil
			if initialized {
				e.runCleanup()
			}

			return effect()
		}),

		typ: typ,
	}
	e.fn = e.run

	e.OnCleanup(func() { e.runCleanup() })

	return e
}

func (e *Effect) run() {
	r := GetRuntime()

	r.effectQueue.Enqueue(e.typ, e.Computed.run)
	r.Schedule()
}

func (e *Effect) runCleanup() {
	cleanup := e.Value()
	if fn, ok := cleanup.(func()); ok && fn != nil {
		fn()
	}
}
