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
	c := r.NewComputed(func() any { // an effect is just a computed that returns a cleanup function
		return effect()
	})
	compute := c.fn

	e := &Effect{
		Computed: c,
		typ:      typ,
	}
	e.fn = func() {
		r.effectQueue.Enqueue(typ, func() {
			cleanup := e.Value().(func())

			if cleanup != nil {
				cleanup()
			}

			compute()
		})
	}

	return e
}
