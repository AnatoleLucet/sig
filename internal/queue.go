package internal

type EffectQueue struct {
	effects map[EffectType][]func()
}

func NewEffectQueue() *EffectQueue {
	effects := make(map[EffectType][]func())
	effects[EffectRender] = make([]func(), 0)
	effects[EffectUser] = make([]func(), 0)

	return &EffectQueue{effects}
}

func (q *EffectQueue) Enqueue(typ EffectType, fn func()) {
	q.effects[typ] = append(q.effects[typ], fn)
}

func (q *EffectQueue) RunEffects(typ EffectType) {
	effects := q.effects[typ]
	q.ClearEffects(typ)

	for _, effect := range effects {
		effect()
	}
}

func (q *EffectQueue) ClearEffects(typ EffectType) {
	q.effects[typ] = q.effects[typ][:0]
}

type NodeQueue struct {
	signals []*Signal
}

func NewNodeQueue() *NodeQueue {
	return &NodeQueue{
		signals: make([]*Signal, 0),
	}
}

func (q *NodeQueue) Enqueue(node *Signal) {
	q.signals = append(q.signals, node)
}

func (q *NodeQueue) Commit() {
	for _, node := range q.signals {
		node.Commit()
	}

	q.signals = q.signals[:0]
}

type SettledQueue struct {
	callbacks []func()
}

func NewSettledQueue() *SettledQueue {
	return &SettledQueue{
		callbacks: make([]func(), 0),
	}
}

func (q *SettledQueue) Enqueue(fn func()) {
	q.callbacks = append(q.callbacks, fn)
}

func (q *SettledQueue) Run() {
	callbacks := q.callbacks
	q.callbacks = q.callbacks[:0]

	for _, cb := range callbacks {
		cb()
	}
}
