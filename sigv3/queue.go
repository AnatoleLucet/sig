package sig

type EffectType int

const (
	EffectRender EffectType = iota
	EffectUser
)

type EffectQueue struct {
	renderEffects []func()
	userEffects   []func()
}

func NewQueue() *EffectQueue {
	return &EffectQueue{
		renderEffects: make([]func(), 0),
		userEffects:   make([]func(), 0),
	}
}

func (q *EffectQueue) Enqueue(typ EffectType, fn func()) {
	switch typ {
	case EffectRender:
		q.renderEffects = append(q.renderEffects, fn)
	case EffectUser:
		q.userEffects = append(q.userEffects, fn)
	}
}

func (q *EffectQueue) RunEffects(typ EffectType) {
	effects := q.getEffects(typ)
	q.ClearEffects(typ)

	for _, effect := range effects {
		effect()
	}

	// TODO: recursively run children children
}

func (q *EffectQueue) ClearEffects(typ EffectType) {
	switch typ {
	case EffectRender:
		q.renderEffects = q.renderEffects[:0]
	case EffectUser:
		q.userEffects = q.userEffects[:0]
	}
}

func (q *EffectQueue) getEffects(typ EffectType) []func() {
	switch typ {
	case EffectRender:
		return q.renderEffects
	case EffectUser:
		return q.userEffects
	}

	return nil
}
