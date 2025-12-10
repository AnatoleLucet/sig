package internal

type EffectType int

const (
	EffectRender EffectType = iota
	EffectUser
)

type Effect struct {
	*Computed // an effect is just a computed that returns a cleanup function

	typ EffectType
}

func (r *Runtime) NewEffect(typ EffectType, effect func() func()) *Effect {
	c := r.NewComputed(func() any {
		return effect()
	})
	computeFn := c.fn

	e := &Effect{
		Computed: c,
		typ:      typ,
	}
	e.fn = func() {
		computeFn()

		// cleanup := e.Value()

		// enqueue the effect for execution
	}

	return e
}
