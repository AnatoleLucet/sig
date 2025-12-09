package sig

type EffectType int

const (
	EffectRender EffectType = iota
	EffectUser
)

type EffectQueue struct {
	parent   *EffectQueue
	children []*EffectQueue

	renderEffects []func()
	userEffects   []func()
}

func NewQueue() *EffectQueue {
	return &EffectQueue{
		renderEffects: make([]func(), 0),
		userEffects:   make([]func(), 0),
	}
}

func (q *EffectQueue) AddChild(child *EffectQueue) {
	q.children = append(q.children, child)
	child.parent = q
}

func (q *EffectQueue) RemoveChild(child *EffectQueue) {
	for i, c := range q.children {
		if c == child {
			q.children = append(q.children[:i], q.children[i+1:]...)
			child.parent = nil
			break
		}
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

	for _, child := range q.children {
		child.RunEffects(typ)
	}
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
