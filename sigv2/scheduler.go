package sigv2

import (
	"fmt"
	"sync"

	"github.com/petermattis/goid"
)

var (
	clock            uint64
	scheduled        bool
	schedulerMutex   sync.Mutex
	staleFlag        sync.Map // map[int64]bool - per goroutine stale state
	batchDepthMap    sync.Map // map[int64]int - per goroutine batch depth
	flushWaiters     []chan struct{}
	flushWaitersMutex sync.Mutex
	dirtyHeap        *heap
	pendingHeap      *heap
	globalQueue      *Queue
	pendingNodesList *pendingNodes
)

func init() {
	dirtyHeap = newHeap(2000)
	pendingHeap = newHeap(2000)
	globalQueue = &Queue{
		queues: [2][]func(EffectType){},
	}
	globalQueue.created = clock
	pendingNodesList = &pendingNodes{
		nodes: make([]interface{}, 0, 100),
	}
}

// getClock returns the current logical clock
func getClock() uint64 {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	return clock
}

// getDirtyHeap returns the dirty heap
func getDirtyHeap() *heap {
	return dirtyHeap
}

// getPendingHeap returns the pending heap
func getPendingHeap() *heap {
	return pendingHeap
}

// getGlobalQueue returns the global queue
func getGlobalQueue() *Queue {
	return globalQueue
}

// getPendingNodes returns the pending nodes list
func getPendingNodes() *pendingNodes {
	return pendingNodesList
}

// isStale returns whether we're currently reading stale values
func isStale() bool {
	gid := getGoroutineID()
	if stale, ok := staleFlag.Load(gid); ok {
		return stale.(bool)
	}
	return false
}

// setStale sets the stale flag for the current goroutine
func setStale(value bool) {
	gid := getGoroutineID()
	staleFlag.Store(gid, value)
}

// getGoroutineID returns the current goroutine ID
func getGoroutineID() int64 {
	return goid.Get()
}

// getBatchDepth returns the current batch depth for this goroutine
func getBatchDepth() int {
	gid := getGoroutineID()
	if depth, ok := batchDepthMap.Load(gid); ok {
		return depth.(int)
	}
	return 0
}

// incrementBatchDepth increases batch depth and returns new depth
func incrementBatchDepth() int {
	gid := getGoroutineID()
	depth := getBatchDepth() + 1
	batchDepthMap.Store(gid, depth)
	return depth
}

// decrementBatchDepth decreases batch depth
func decrementBatchDepth() {
	gid := getGoroutineID()
	depth := getBatchDepth() - 1
	if depth <= 0 {
		batchDepthMap.Delete(gid)
	} else {
		batchDepthMap.Store(gid, depth)
	}
}

// pendingNodes tracks nodes with pending value updates
type pendingNodes struct {
	mu    sync.Mutex
	nodes []interface{} // Can be any baseNode type
}

func (p *pendingNodes) add(node interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if already added
	for _, n := range p.nodes {
		if n == node {
			return
		}
	}

	p.nodes = append(p.nodes, node)
}

func (p *pendingNodes) clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nodes = p.nodes[:0]
}

func (p *pendingNodes) get() []interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]interface{}{}, p.nodes...)
}

func (p *pendingNodes) len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.nodes)
}

// Queue manages effect execution and scheduling
type Queue struct {
	mu       sync.Mutex
	parent   *Queue
	running  bool
	queues   [2][]func(EffectType)
	children []*Queue
	created  uint64
}

// enqueue adds an effect to the queue
func (q *Queue) enqueue(effectType EffectType, fn func(EffectType)) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if effectType > 0 && effectType <= 2 {
		q.queues[effectType-1] = append(q.queues[effectType-1], fn)
	}

	schedule()
}

// run executes all queued effects of a given type
func (q *Queue) run(effectType EffectType) {
	q.mu.Lock()
	effects := q.queues[effectType-1]
	q.queues[effectType-1] = nil
	q.mu.Unlock()

	if len(effects) > 0 {
		for _, effect := range effects {
			effect(effectType)
		}
	}

	// Run children's queues
	q.mu.Lock()
	children := append([]*Queue{}, q.children...)
	q.mu.Unlock()

	for _, child := range children {
		child.run(effectType)
	}
}

// flush processes all pending updates and effects
func (q *Queue) flush() {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.mu.Unlock()

	defer func() {
		q.mu.Lock()
		q.running = false
		q.mu.Unlock()
	}()

	// Run dirty heap
	runHeap(dirtyHeap, func(node heapNode) {
		if recomputable, ok := node.(Recomputable); ok {
			recomputable.recompute(false)
		}
	})

	// Handle pending nodes
	nodes := pendingNodesList.get()
	for _, nodeInterface := range nodes {
		// Use Committable interface to commit pending values
		if committable, ok := nodeInterface.(Committable); ok {
			committable.commitPendingValue()
		}
	}

	pendingNodesList.clear()

	// Increment clock
	schedulerMutex.Lock()
	clock++
	scheduled = false
	schedulerMutex.Unlock()

	// Run effects
	q.run(EffectTypeRender)
	q.run(EffectTypeUser)
}

// schedule queues a flush operation
func schedule() {
	// Don't schedule during batch operations
	if getBatchDepth() > 0 {
		return
	}

	schedulerMutex.Lock()
	if scheduled {
		schedulerMutex.Unlock()
		return
	}

	scheduled = true
	schedulerMutex.Unlock()

	// Schedule flush asynchronously to avoid deadlocks
	go flush()
}

// waitForFlush blocks until all pending updates have been processed
func waitForFlush() {
	// Check if there's anything to wait for
	schedulerMutex.Lock()
	if !scheduled && getPendingNodes().len() == 0 && getDirtyHeap().max == 0 {
		schedulerMutex.Unlock()
		return // Nothing to wait for
	}
	schedulerMutex.Unlock()

	// Create a channel to wait on
	done := make(chan struct{})

	flushWaitersMutex.Lock()
	flushWaiters = append(flushWaiters, done)
	flushWaitersMutex.Unlock()

	// Wait for notification
	<-done
}

// flush processes all updates with infinite loop detection
func flush() {
	defer func() {
		// Notify all waiters that flush is complete
		flushWaitersMutex.Lock()
		waiters := flushWaiters
		flushWaiters = nil
		flushWaitersMutex.Unlock()

		for _, ch := range waiters {
			close(ch)
		}
	}()

	count := 0
	for {
		schedulerMutex.Lock()
		shouldContinue := scheduled
		schedulerMutex.Unlock()

		if !shouldContinue {
			break
		}

		count++
		if count >= 100000 {
			panic(fmt.Errorf("Potential Infinite Loop Detected"))
		}

		globalQueue.flush()
	}
}

// staleValues executes a function while reading stale values
func staleValues[T any](fn func() T, setFlag bool) T {
	prevStale := isStale()
	setStale(setFlag)
	defer setStale(prevStale)

	return fn()
}
