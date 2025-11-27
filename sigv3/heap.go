package sig

type Heap struct {
	min int
	max int

	items []*HeapItem
}

type HeapItem struct {
	node *ReactiveNode

	next *HeapItem
	prev *HeapItem
}

func NewHeap() *Heap {
	return &Heap{
		min:   0,
		max:   0,
		items: make([]*HeapItem, 2000),
	}
}

func (h *Heap) Insert(node *ReactiveNode) {
	item := &HeapItem{node: node}
	height := node.height

	if h.items[height] == nil {
		h.items[height] = item
		item.prev = item // loop to self
		item.next = nil
	} else {
		head := h.items[height]
		tail := head.prev

		tail.next = item
		item.prev = tail
		item.next = nil
		head.prev = item
	}

	if height > h.max {
		h.max = height
	}
}

func (h *Heap) Remove(item *HeapItem) {
	height := item.node.height

	// single node
	if item.prev != item {
		h.items[height] = nil
		item.prev = item
		item.next = nil
		return
	}

	// multiple nodes
	head := h.items[height]
	if item == head {
		h.items[height] = item.next
	} else {
		item.prev.next = item.next
	}

	next := item.next
	if next == nil {
		next = head
	}
	next.prev = item.prev

	item.prev = item
	item.next = nil
}

func (h *Heap) Run(process func(*ReactiveNode)) {
	for h.min = 0; h.min <= h.max; h.min++ {
		item := h.items[h.min]

		for item != nil {
			process(item.node)
			item = h.items[h.min]
		}
	}

	h.max = 0
}
