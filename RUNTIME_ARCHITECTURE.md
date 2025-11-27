# Ryan v2 Go Implementation - Runtime Architecture

This document outlines the architecture for a concurrency-safe Go implementation of Ryan v2's reactive system.

## Table of Contents

1. [Overview](#overview)
2. [Core Challenge](#core-challenge)
3. [Concurrency Strategy](#concurrency-strategy)
4. [Structure Decomposition](#structure-decomposition)
5. [Component Details](#component-details)
6. [Runtime Coordinator](#runtime-coordinator)
7. [Public API](#public-api)
8. [Benefits](#benefits)
9. [Alternative Approaches](#alternative-approaches)

---

## Overview

Ryan v2 is a fine-grained reactive system originally designed for single-threaded JavaScript. This architecture adapts it for Go while maintaining:

- Fine-grained reactivity
- Topological execution order (via heaps)
- Batched updates
- Async transitions
- Effect scheduling

---

## Core Challenge

Ryan v2 assumes **single-threaded execution**. Key assumptions that break with concurrency:

1. **Synchronous execution**: No locks needed in JavaScript
2. **Global logical clock**: Single source of truth for ordering
3. **Batched updates**: Microtask queue guarantees sequential processing
4. **Height-based topological sort**: Assumes no concurrent modifications during traversal

---

## Concurrency Strategy

### Per-Runtime Isolation

Each `Runtime` is **single-threaded by design**. For concurrency, create multiple isolated Runtimes:

```go
// Option A: Per-goroutine isolation
runtime := GetRuntimeForCurrentGoroutine()

// Option B: Explicit Runtime per "reactive scope"
runtime1 := NewRuntime()  // UI thread
runtime2 := NewRuntime()  // Worker thread
```

### Locking Strategy

- **Single lock per Runtime**: One `sync.Mutex` protects entire reactive graph
- **No deadlocks**: Only one lock in the system
- **Trade-off**: Coarse-grained, but reactive systems execute quickly anyway

### Per-Goroutine Runtime Registry

```go
// Global registry of runtimes per goroutine
var runtimes sync.Map  // goroutine ID → *Runtime

// GetRuntime returns the Runtime for the current goroutine.
// Creates a new one if it doesn't exist.
func GetRuntime() *Runtime {
    gid := goid.Get()

    if rt, ok := runtimes.Load(gid); ok {
        return rt.(*Runtime)
    }

    rt := NewRuntime()
    runtimes.Store(gid, rt)
    return rt
}
```

---

## Structure Decomposition

The monolithic Runtime struct is split into focused components following the **single responsibility principle**:

```go
// Runtime is the main coordinator that owns everything
type Runtime struct {
    mu sync.Mutex

    scheduler *Scheduler
    graph     *ReactiveGraph
    context   *ExecutionContext
    asyncMgr  *AsyncManager
}
```

---

## Component Details

### 1. Scheduler - "When and how do updates happen?"

```go
// Scheduler handles batching, effect execution, and flush coordination
type Scheduler struct {
    clock     int
    scheduled bool

    dirtyHeap   *Heap   // Nodes to recompute
    pendingHeap *Heap   // Zombies/async nodes

    globalQueue *Queue  // Effect coordination
}

func (s *Scheduler) Schedule() {
    if s.scheduled {
        return
    }
    s.scheduled = true
    // ... queue microtask
}

func (s *Scheduler) Flush(
    graph *ReactiveGraph,
    ctx *ExecutionContext,
    async *AsyncManager,
) {
    // Coordinate heaps, queues, transitions
    s.runDirtyHeap()
    async.HandleTransitions(s)
    s.commitPendingValues()
    s.globalQueue.RunEffects()
}
```

**Responsibilities:**
- Clock management
- Heap coordination
- Batching logic
- Determining when to flush

---

### 2. Queue - "How do effects execute in order?"

```go
// Queue manages effect execution and hierarchical scopes
type Queue struct {
    parent   *Queue
    children []*Queue

    running       bool
    renderEffects []func()
    userEffects   []func()
    pendingNodes  []*ReactiveNode
    created       int  // Clock tick when created
}

func (q *Queue) Enqueue(effectType EffectType, fn func()) {
    switch effectType {
    case EffectRender:
        q.renderEffects = append(q.renderEffects, fn)
    case EffectUser:
        q.userEffects = append(q.userEffects, fn)
    }
}

func (q *Queue) RunEffects(effectType EffectType) {
    // Run this queue's effects
    effects := q.getEffects(effectType)
    q.clearEffects(effectType)

    for _, effect := range effects {
        effect()
    }

    // Recursively run children
    for _, child := range q.children {
        child.RunEffects(effectType)
    }
}

func (q *Queue) AddChild(child *Queue) {
    q.children = append(q.children, child)
    child.parent = q
}
```

**Responsibilities:**
- Hierarchical effect management
- Render vs user effect ordering
- Parent/child queue scopes (for component boundaries)

**Why separate from Scheduler?**
- Scheduler decides **when** to run effects
- Queue decides **which** effects and **in what order**
- Queues can be nested (child components), Scheduler is singular

---

### 3. ReactiveGraph - "How are nodes connected?"

```go
// ReactiveGraph manages the dependency graph and node lifecycle
type ReactiveGraph struct {
    nodes      []*ReactiveNode  // Track all nodes for GC, debugging
    unobserved []*ReactiveNode  // Signals that lost all observers
}

func (g *ReactiveGraph) Link(dep, sub *ReactiveNode) {
    // Create bidirectional dependency link
    // (Same logic as Ryan's link function)
}

func (g *ReactiveGraph) Unlink(link *DependencyLink) {
    // Remove dependency link
}

func (g *ReactiveGraph) MarkSubscribers(node *ReactiveNode, state ReactiveFlags) {
    // Propagate dirty/check flags
    for link := node.subs; link != nil; link = link.nextSub {
        g.markNode(link.sub, state)
    }
}

func (g *ReactiveGraph) TrackUnobserved(node *ReactiveNode) {
    if node.subs == nil && node.unobserved != nil {
        g.unobserved = append(g.unobserved, node)
    }
}

func (g *ReactiveGraph) NotifyUnobserved() {
    for _, node := range g.unobserved {
        if node.subs == nil {
            node.unobserved()
        }
    }
    g.unobserved = g.unobserved[:0]
}
```

**Responsibilities:**
- Dependency tracking (link/unlink)
- Graph traversal (mark subscribers)
- Unobserved signal notifications
- Node lifecycle (register all nodes for debugging)

---

### 4. ExecutionContext - "What's currently running?"

```go
// ExecutionContext tracks what's currently executing
type ExecutionContext struct {
    currentNode *ReactiveNode  // Currently executing computed/effect
    tracking    bool           // Whether to track dependencies

    // For special read modes
    stale            bool
    pendingValueCheck bool
    pendingCheck      *struct{ value bool }
}

func (ctx *ExecutionContext) WithNode(node *ReactiveNode, fn func()) {
    prev := ctx.currentNode
    ctx.currentNode = node
    defer func() { ctx.currentNode = prev }()

    fn()
}

func (ctx *ExecutionContext) WithTracking(enabled bool, fn func()) any {
    prev := ctx.tracking
    ctx.tracking = enabled
    defer func() { ctx.tracking = prev }()

    return fn()
}

func (ctx *ExecutionContext) ShouldTrack(node *ReactiveNode) bool {
    return ctx.currentNode != nil && ctx.tracking
}
```

**Responsibilities:**
- Track current execution context
- Control dependency tracking on/off
- Manage stale/pending read modes

---

### 5. AsyncManager - "How do async operations coordinate?"

```go
// AsyncManager handles transitions and async operations
type AsyncManager struct {
    activeTransition *Transition
    transitions      []*Transition
}

type Transition struct {
    time         int
    pendingNodes []*ReactiveNode
    asyncNodes   []*ReactiveNode
    queues       struct {
        render []func()
        user   []func()
    }
}

func (am *AsyncManager) InitTransition(node *ReactiveNode, scheduler *Scheduler) {
    if am.activeTransition != nil && am.activeTransition.time == scheduler.clock {
        return
    }

    if am.activeTransition == nil {
        am.activeTransition = &Transition{
            time: scheduler.clock,
            pendingNodes: make([]*ReactiveNode, 0),
            asyncNodes: make([]*ReactiveNode, 0),
        }
    }

    am.transitions = append(am.transitions, am.activeTransition)
    am.activeTransition.time = scheduler.clock

    // Move pending nodes to transition
    for _, n := range scheduler.globalQueue.pendingNodes {
        n.transition = am.activeTransition
        am.activeTransition.pendingNodes = append(am.activeTransition.pendingNodes, n)
    }

    scheduler.globalQueue.pendingNodes = am.activeTransition.pendingNodes
}

func (am *AsyncManager) IsTransitionComplete(t *Transition) bool {
    for _, node := range t.asyncNodes {
        if node.statusFlags&StatusPending != 0 {
            return false
        }
    }
    return true
}

func (am *AsyncManager) HandleTransitions(scheduler *Scheduler) {
    if am.activeTransition == nil {
        return
    }

    if !am.IsTransitionComplete(am.activeTransition) {
        // Suspend transition
        scheduler.pendingHeap.Run(func(item *HeapItem) {
            // recompute...
        })
        // ... suspend logic
        am.activeTransition = nil
        return
    }

    // Merge completed transition
    scheduler.globalQueue.pendingNodes = append(
        scheduler.globalQueue.pendingNodes,
        am.activeTransition.pendingNodes...,
    )
    // ... merge effects

    am.removeTransition(am.activeTransition)
    am.activeTransition = nil
}
```

**Responsibilities:**
- Manage async transitions (Promises/Suspense)
- Track pending async operations
- Coordinate transition completion

---

## Runtime Coordinator

The Runtime owns all components and provides the public API:

```go
type Runtime struct {
    mu sync.Mutex

    scheduler *Scheduler
    graph     *ReactiveGraph
    context   *ExecutionContext
    asyncMgr  *AsyncManager
}

func NewRuntime() *Runtime {
    scheduler := &Scheduler{
        dirtyHeap:   NewHeap(),
        pendingHeap: NewHeap(),
        globalQueue: &Queue{
            renderEffects: make([]func(), 0),
            userEffects:   make([]func(), 0),
            pendingNodes:  make([]*ReactiveNode, 0),
        },
    }

    return &Runtime{
        scheduler: scheduler,
        graph:     &ReactiveGraph{},
        context:   &ExecutionContext{tracking: true},
        asyncMgr:  &AsyncManager{},
    }
}

// === Public API delegates to components ===

func (r *Runtime) CreateSignal(initial any, opts ...Option) *ReactiveNode {
    r.mu.Lock()
    defer r.mu.Unlock()

    node := &ReactiveNode{
        value:        initial,
        pendingValue: notPending,
        time:         r.scheduler.clock,
        // ...
    }

    r.graph.nodes = append(r.graph.nodes, node)
    return node
}

func (r *Runtime) Read(node *ReactiveNode) any {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Track dependency if in reactive context
    if r.context.ShouldTrack(node) {
        r.graph.Link(node, r.context.currentNode)
    }

    // Update if necessary
    r.updateIfNecessary(node)

    // Return value (or pendingValue)
    return r.getValue(node)
}

func (r *Runtime) Write(node *ReactiveNode, value any) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if !r.valueChanged(node, value) {
        return
    }

    // Set pending value
    if node.optimistic {
        node.value = value
    } else {
        if node.pendingValue == notPending {
            r.scheduler.globalQueue.pendingNodes = append(
                r.scheduler.globalQueue.pendingNodes,
                node,
            )
        }
        node.pendingValue = value
    }

    node.time = r.scheduler.clock

    // Mark subscribers dirty
    r.graph.MarkSubscribers(node, FlagDirty)

    // Insert into dirty heap
    for link := node.subs; link != nil; link = link.nextSub {
        heap := r.scheduler.dirtyHeap
        if link.sub.flags&FlagZombie != 0 {
            heap = r.scheduler.pendingHeap
        }
        heap.Insert(link.sub)
    }

    r.scheduler.Schedule()
}

func (r *Runtime) Flush() {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.scheduler.Flush(r.graph, r.context, r.asyncMgr)
}

func (r *Runtime) Untrack(fn func() any) any {
    r.mu.Lock()
    defer r.mu.Unlock()

    return r.context.WithTracking(false, fn)
}
```

---

## Public API

Clean API that matches Ryan v2's interface:

```go
// Signal creates a reactive value
func Signal[T any](initial T) (get func() T, set func(T)) {
    rt := GetRuntime()
    node := rt.CreateSignal(initial)

    return func() T {
        return rt.Read(node).(T)
    }, func(val T) {
        rt.Write(node, val)
    }
}

// Memo creates a computed value
func Memo[T any](fn func() T) func() T {
    rt := GetRuntime()
    node := rt.CreateMemo(func() any { return fn() })

    return func() T {
        return rt.Read(node).(T)
    }
}

// Effect creates a side effect
func Effect(fn func()) {
    rt := GetRuntime()
    rt.CreateEffect(
        func() any { fn(); return nil },
        func(_, _ any) func() { return nil },
    )
}

// Batch batches multiple updates
func Batch(fn func()) {
    rt := GetRuntime()
    rt.Batch(fn)
}

// Untrack reads without tracking dependencies
func Untrack[T any](fn func() T) T {
    rt := GetRuntime()
    return rt.Untrack(func() any { return fn() }).(T)
}
```

---

## Data Structures

### ReactiveNode

```go
// ReactiveNode represents any reactive value (signal, computed, effect)
type ReactiveNode struct {
    // === Value state ===
    value        any
    pendingValue any  // NOT_PENDING sentinel when no pending update
    equals       func(a, b any) bool

    // === Dependency graph (bidirectional links) ===
    deps     *DependencyLink
    depsTail *DependencyLink
    subs     *DependencyLink
    subsTail *DependencyLink

    // === Heap membership ===
    heapItem *HeapItem  // Reference to heap item (nil if not in heap)
    height   int        // Topological depth

    // === Tree structure (parent/child for disposal) ===
    parent       *ReactiveNode
    nextSibling  *ReactiveNode
    firstChild   *ReactiveNode

    // === Disposal ===
    disposal            any  // Function or []Function
    pendingDisposal     any
    pendingFirstChild   *ReactiveNode

    // === Execution state ===
    fn    func(prevValue any) any  // Computation function (nil for signals)
    flags ReactiveFlags
    time  int  // Last update time (clock tick)

    // === Async state ===
    statusFlags StatusFlags
    err         error
    transition  *Transition

    // === Metadata ===
    name       string
    id         string
    childCount int

    // === Queue reference ===
    queue *Queue  // For hierarchical queue scopes

    // === Context ===
    context map[any]any

    // === Options ===
    optimistic bool  // Updates immediately without batching
    pureWrite  bool
    unobserved func()

    // === Root-specific ===
    root           bool
    parentComputed *ReactiveNode

    // === Effect-specific ===
    effectType  EffectType
    modified    bool
    prevValue   any
    effectFn    func(value, prevValue any) (cleanup func())
    errorFn     func(err error, reset func())
    cleanup     func()
    notifyQueue func()

    // === Optimistic updates ===
    pendingCheck  *ReactiveNode
    pendingSignal *ReactiveNode

    // === Firewall (nested signals) ===
    owner     *ReactiveNode
    nextChild *ReactiveNode
    child     *ReactiveNode
}
```

### DependencyLink

```go
type DependencyLink struct {
    dep     *ReactiveNode
    sub     *ReactiveNode
    nextDep *DependencyLink
    prevSub *DependencyLink
    nextSub *DependencyLink
}
```

### Flags and Enums

```go
type ReactiveFlags int

const (
    FlagNone ReactiveFlags = 0
    FlagCheck ReactiveFlags = 1 << iota
    FlagDirty
    FlagRecomputingDeps
    FlagInHeap
    FlagInHeapHeight
    FlagZombie
)

type StatusFlags int

const (
    StatusNone StatusFlags = 0
    StatusPending StatusFlags = 1 << iota
    StatusError
    StatusUninitialized
)

type EffectType int

const (
    EffectRender EffectType = 1
    EffectUser   EffectType = 2
)
```

### Sentinel Values

```go
// Sentinel value for "no pending update"
var notPending = &struct{}{}
```

---

## Benefits

### 1. Clear Responsibilities

Each struct has one focused job:
- `Runtime`: Owns everything, provides public API
- `Scheduler`: Timing and batching
- `Queue`: Effect ordering
- `ReactiveGraph`: Dependency management
- `ExecutionContext`: Execution state
- `AsyncManager`: Async coordination

### 2. Easier Testing

```go
func TestSchedulerFlush(t *testing.T) {
    scheduler := &Scheduler{...}
    graph := &ReactiveGraph{}
    ctx := &ExecutionContext{}
    async := &AsyncManager{}

    // Test scheduler in isolation
    scheduler.Flush(graph, ctx, async)
}
```

### 3. Better Encapsulation

- Queue doesn't know about heaps
- Scheduler doesn't know about dependency links
- Graph doesn't know about async transitions

### 4. Matches Ryan's Conceptual Model

Ryan v2 has these same concepts, just implemented as global functions/variables in TypeScript.

---

## Alternative Approaches

### Alternative 1: Flatter Structure

If you want **less nesting** but still organized:

```go
type Runtime struct {
    mu sync.Mutex

    // Scheduling
    clock         int
    scheduled     bool
    dirtyHeap     *Heap
    pendingHeap   *Heap

    // Effects (queue replaces global arrays)
    queue *Queue

    // Graph
    unobserved []*ReactiveNode

    // Context
    currentNode       *ReactiveNode
    tracking          bool
    stale             bool
    pendingValueCheck bool
    pendingCheck      *struct{ value bool }

    // Async
    activeTransition *Transition
    transitions      []*Transition
}
```

Then extract **behavior** into separate files:
- `runtime_scheduler.go` - scheduling methods
- `runtime_graph.go` - graph traversal methods
- `runtime_async.go` - async transition methods
- `runtime_api.go` - public API methods

### Alternative 2: Interface-Based (Maximum Decoupling)

```go
// Scheduler interface
type Scheduler interface {
    MarkDirty(node *ReactiveNode)
    Flush()
    IsScheduled() bool
}

// Heap interface
type PriorityQueue interface {
    Insert(node *ReactiveNode)
    Remove(item *HeapItem)
    Process(fn func(*ReactiveNode))
}

// Runtime implements Scheduler
type Runtime struct {
    dirtyQueue   PriorityQueue
    pendingQueue PriorityQueue
    // ...
}

// Now you can swap implementations!
```

---

## Recommendation

**Start with the separate structs approach** (main approach in this document):

1. **Simple**: One Runtime struct owns everything
2. **Testable**: Easy to create isolated instances
3. **Go-idiomatic**: Clear struct ownership, no global state
4. **Maintainable**: Easy to see what depends on what

Avoid over-abstracting with interfaces initially. Go's philosophy is to start concrete and extract interfaces only when you have multiple implementations.

---

## Example: Async Handling in Go

Ryan's async computed uses Promises. Go equivalent uses goroutines + channels:

```go
func AsyncMemo[T any](fn func() T) func() T {
    rt := GetRuntime()

    // Launch async work
    resultChan := make(chan T)
    go func() {
        result := fn()
        resultChan <- result
    }()

    // Create suspended memo
    node := rt.CreateAsyncComputed(func() any {
        select {
        case result := <-resultChan:
            return result
        default:
            // Not ready - throw NotReadyError equivalent
            panic(NotReadyError{})
        }
    })

    return func() T {
        return rt.Read(node).(T)
    }
}
```

---

## Concurrency Safety Notes

### Critical Decisions

1. **Single Lock Per Runtime**
   - Simple: One `sync.Mutex` protects entire reactive graph
   - No deadlocks: Only one lock in the system
   - Trade-off: Coarse-grained, but reactive systems are fast anyway

2. **Per-Goroutine Isolation**
   - Each goroutine gets its own Runtime
   - No cross-goroutine interference
   - Explicit sharing if needed (channels, sync primitives)

3. **Lock Held During Entire Operations**
   - `Read()`, `Write()`, `Flush()` all acquire lock at start
   - Released only when operation completes
   - User code (effect functions, computations) runs while holding lock
   - This is acceptable because reactive operations should be fast

### When to Use Multiple Runtimes

- **UI thread**: Main runtime for UI reactivity
- **Worker threads**: Separate runtime per worker
- **Request handlers**: Runtime per request (if using goroutines per request)

### Sharing Data Across Runtimes

If you need to share reactive state across goroutines:

```go
// DON'T: Share nodes across runtimes (race conditions)
node := runtime1.CreateSignal(5)
runtime2.Read(node)  // UNSAFE!

// DO: Use channels or other Go primitives
runtime1 := NewRuntime()
runtime2 := NewRuntime()

ch := make(chan int)

// Runtime 1
sig1 := runtime1.CreateSignal(5)
runtime1.CreateEffect(func() {
    val := runtime1.Read(sig1)
    ch <- val.(int)
})

// Runtime 2
sig2 := runtime2.CreateSignal(0)
go func() {
    for val := range ch {
        runtime2.Write(sig2, val)
    }
}()
```

---

## File Organization

Suggested package structure:

```
sig/
├── runtime.go           // Runtime struct and main coordinator
├── scheduler.go         // Scheduler implementation
├── queue.go             // Queue implementation
├── graph.go             // ReactiveGraph implementation
├── context.go           // ExecutionContext implementation
├── async.go             // AsyncManager implementation
├── node.go              // ReactiveNode struct definition
├── heap.go              // Heap data structure
├── api.go               // Public API (Signal, Memo, Effect, etc.)
├── options.go           // Option pattern for configuration
└── errors.go            // Custom error types
```

---

## Next Steps

1. Implement the `Heap` data structure (already done in `sigv3/heap.go`)
2. Implement `ReactiveNode` and `DependencyLink` structs
3. Implement each component (`Scheduler`, `Queue`, `ReactiveGraph`, etc.)
4. Implement `Runtime` coordinator
5. Implement public API
6. Write tests for each component
7. Write integration tests for the full system
8. Benchmark and optimize

---

## References

- [Ryan v2 TypeScript Implementation](../ryanv2/reactivity-readable.ts)
- [Ryan v2 Minified](../ryanv2/reactivity.ts)
- [Original sig/ Implementation](../sig/)
