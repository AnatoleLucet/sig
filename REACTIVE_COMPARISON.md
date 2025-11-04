# Reactive System Comparison: Ryan vs sig/

## Overview

This document compares Ryan's TypeScript reactive implementation with the current Go implementation in `sig/`, and provides a roadmap for bringing production-ready features to the Go version.

---

## Ryan's TypeScript Implementation

A **highly sophisticated** reactive system with several advanced features.

### Core Architecture

- **Heap-based scheduling**: Uses a priority heap organized by "height" (dependency depth) - processes nodes from lowest to highest height to ensure parents run before children
- **Bitflag state tracking**: Uses flags (Check, Dirty, RecomputingDeps, InHeap, Zombie, etc.) to efficiently track node states
- **Height tracking**: Each computed node has a dynamic height based on dependency depth, ensuring topological ordering
- **Pending value system**: Changes are buffered in `pendingValue` and committed together during stabilization
- **Clock versioning**: Uses a global `clock` variable for version tracking

### Advanced Features

- **Full async support**: Built-in handling for Promises and async iterators via `asyncComputed`
  - `NotReadyError` for pending values
  - Transition system for managing async state changes
  - Separate async flags (Pending, Error, Uninitialized)
- **Zombie nodes**: Nodes marked for disposal but still referenced are moved to a separate `pending` heap
- **Glitch prevention**: The height-based heap ensures no inconsistent intermediate states
- **Smart dependency tracking**: Intrusive doubly-linked lists for efficient dependency/subscription management

### How It Works

1. When a signal changes, it marks subscribers and inserts them into the heap
2. `stabilize()` processes the dirty heap from lowest to highest height
3. During recomputation, dependencies are tracked and old ones removed
4. Async nodes trigger transitions with separate pending heap processing
5. Pending values are committed after stabilization

---

## Current Go Implementation (sig/)

A **much simpler** reactive system focused on clarity.

### Core Architecture

- **Immediate execution**: Reactions execute right away (or queue if batching)
- **Simple tracking**: Uses basic slices for `reactions[]` and `dependencies[]`
- **Boolean dirty flag**: Memos use a single `dirty` bool for invalidation
- **No scheduling**: No heap, no height tracking, no topological ordering

### Features

- **Batch mode**: Simple depth counter to queue reactions during batches
- **Ownership tree**: Parent/child relationships for disposal (similar to ryan)
- **Thread-safety**: Uses `sync.Map` and mutexes with per-goroutine owner tracking (via `goid` library)

### How It Works

1. When a signal changes, it notifies all reactions
2. If not batching, reactions execute immediately
3. If batching, reactions are queued and executed when batch ends
4. Memos mark themselves dirty and recompute on next `Get()`
5. No glitch prevention - multiple intermediate states possible

---

## Key Differences Summary

| Feature | Ryan | sig/ |
|---------|------|------|
| **Scheduling** | Heap-based topological sort | Immediate or simple queue |
| **Async support** | Full (Promises, iterators) | None |
| **Glitch prevention** | Yes (height-based ordering) | No |
| **Pending values** | Yes (buffered commits) | No |
| **State tracking** | Bitflags | Simple booleans |
| **Dependency tracking** | Intrusive linked lists | Slices |
| **Complexity** | High (~570 LOC) | Low (~200 LOC) |
| **Thread model** | Single-threaded JS | Multi-threaded Go w/ goroutine tracking |

**Ryan's system is optimized for**:
- Preventing glitches in complex dependency graphs
- Handling async computations elegantly
- Minimizing unnecessary recomputations
- Fine-grained performance optimization

**sig/ is optimized for**:
- Simplicity and readability
- Go's concurrency model
- Straightforward mental model
- Lower overhead for simple use cases

---

## Core Features Worth Bringing to Production Go

### 1. Glitch Prevention via Height-Based Scheduling ⭐ CRITICAL

**Why**: Without this, you can have inconsistent intermediate states.

**Example glitch in current implementation:**

```go
a := Signal(1)
b := Memo(func() { return a() * 2 })  // depends on a
c := Memo(func() { return a() + b() }) // depends on a AND b

// Initial: a=1, b=2, c=3
a.Set(2)

// Without ordering: c might see a=2, b=2 (old) → c=4 (WRONG!)
// With height ordering: b updates first (height 1), then c (height 2) → c=6 (CORRECT)
```

**In Go**: Use the heap pattern, track heights dynamically.

### 2. Dirty/Check Flag Distinction ⭐ IMPORTANT

**Why**: Optimize away unnecessary recomputations.

- `Dirty`: This node definitely needs recomputation
- `Check`: This node *might* need recomputation (check dependencies first)

**In Go**: Use bitflags or separate bools:

```go
type flags uint8

const (
    FlagNone flags = 0
    FlagCheck flags = 1 << iota
    FlagDirty
    FlagInHeap
    FlagRecomputing
)
```

### 3. Pending Value System ⭐ IMPORTANT

**Why**: All updates happen atomically at end of stabilization - no partial observable state.

**In Go**: Same pattern works:

```go
type memo[T any] struct {
    value        T
    pendingValue *T  // nil = not pending
    // ...
}
```

### 4. Smart Dependency Tracking

**Why**: Ryan's intrusive linked lists are more efficient than slicing.

**In Go**: Could use similar pattern or stick with slices (simpler, GC-friendly). Profile first.

---

## What Would Look Different in Go

### 1. Async Support - VERY DIFFERENT

**Ryan's approach**: Built around JS Promises/async iterators

```typescript
asyncComputed(() => fetch(url)) // throws NotReadyError, resolves later
```

**Go approach**: Use channels/goroutines instead

```go
func AsyncMemo[T any](fn func(ctx context.Context) (T, error)) func() (T, error) {
    resultChan := make(chan result[T])

    // Spawn goroutine to compute async
    go func() {
        val, err := fn(context.Background())
        resultChan <- result{val, err}
    }()

    return func() (T, error) {
        // Try non-blocking read
        select {
        case r := <-resultChan:
            return r.value, r.error
        default:
            return zero[T](), ErrNotReady
        }
    }
}
```

Or integrate with existing patterns like `sync/errgroup` or `context.Context`.

### 2. Thread Safety - REQUIRED IN GO

Ryan assumes single-threaded JS. Go needs locks everywhere:

```go
type reactiveContext struct {
    mu sync.RWMutex  // Protect all fields

    heap           *heap
    activeReaction Reaction
    // ...
}

func (rc *reactiveContext) stabilize() {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    // ... stabilization logic
}
```

**Consider**: Per-goroutine reactive contexts (what you already do) vs global context with fine-grained locking.

### 3. Memory Management

**JS**: GC handles everything, object pools less critical

**Go**: Consider:
- `sync.Pool` for frequently allocated nodes
- Pointer vs value semantics (affects escape analysis)
- Careful slice growth to avoid allocations

### 4. No Global State Assumptions

Ryan uses global `context`, `clock`, `dirty`, `pending` heaps.

**Go version should**:
- Make these part of a `Runtime` or `Scheduler` struct
- Support multiple independent reactive graphs
- Be goroutine-safe

```go
type Scheduler struct {
    mu           sync.RWMutex
    clock        uint64
    dirtyHeap    *heap
    pendingHeap  *heap
    pendingNodes []node
}

func NewScheduler() *Scheduler { /* ... */ }
func (s *Scheduler) Stabilize() { /* ... */ }
```

---

## Recommended Architecture for Production Go

```go
// Core scheduling primitives
type Scheduler struct {
    mu          sync.RWMutex
    clock       uint64
    dirty       *minHeap  // sorted by height
    batching    int       // batch depth
    pendingExec []Reaction
}

type node struct {
    height uint32
    flags  flags  // Dirty | Check | InHeap | Recomputing

    // Dependency tracking (intrusive linked list or slices)
    deps     *link
    depsTail *link
    subs     *link
    subsTail *link
}

type signal[T comparable] struct {
    mu    sync.RWMutex
    value T

    // Subscription tracking
    subs     *link
    subsTail *link
}

type memo[T any] struct {
    node  // embed scheduling metadata

    mu           sync.RWMutex
    value        T
    pendingValue *T
    fn           func() T
}
```

---

## Implementation Priority Roadmap

### Phase 1: Core Glitch Prevention

1. ✅ Height-based heap scheduling
2. ✅ Dirty/Check flags optimization
3. ✅ Pending value system
4. ✅ Proper topological execution order

### Phase 2: Performance

1. ✅ Intrusive linked lists for deps/subs (if profiling shows slice overhead)
2. ✅ Better batching (already have basic version)
3. ✅ Object pooling for hot paths

### Phase 3: Go-Specific Features

1. ✅ Goroutine-safe async memo (with channels/context)
2. ✅ Integration with `context.Context` for cancellation
3. ✅ Better error handling (Go style, not throwing)

### Phase 4: Advanced

1. ⚠️ Concurrent stabilization (if benchmarks show need)
2. ⚠️ Lazy evaluation modes
3. ⚠️ Effect scheduling options (immediate/microtask/animation frame equivalents)

---

## What NOT to Bring

- ❌ `NotReadyError` exceptions (use Go errors)
- ❌ JS-style Promises (use channels/contexts)
- ❌ Global mutable state (use explicit Runtime/Scheduler)
- ❌ Zombie/transition complexity (start simpler, add if needed)

---

## Examples of Key Patterns

### Height-Based Heap Example

```go
type minHeap struct {
    heap [][]node  // heap[height] = nodes at that height
    min  int
    max  int
}

func (h *minHeap) insert(n *node) {
    height := n.height

    // Grow heap if needed
    for len(h.heap) <= int(height) {
        h.heap = append(h.heap, nil)
    }

    // Add to circular linked list at this height
    if h.heap[height] == nil {
        h.heap[height] = []node{n}
    } else {
        h.heap[height] = append(h.heap[height], n)
    }

    if height > h.max {
        h.max = height
    }
}

func (s *Scheduler) stabilize() {
    // Process from lowest height to highest
    for h := s.dirty.min; h <= s.dirty.max; h++ {
        for _, node := range s.dirty.heap[h] {
            if node.flags&FlagDirty != 0 {
                s.recompute(node)
            }
        }
    }

    // Commit all pending values atomically
    for _, node := range s.pendingNodes {
        if node.pendingValue != nil {
            node.value = *node.pendingValue
            node.pendingValue = nil
        }
    }
    s.pendingNodes = nil
    s.clock++
}
```

### Dirty/Check Optimization Example

```go
func (s *Scheduler) updateIfNecessary(n *node) {
    // If only marked "check", verify dependencies first
    if n.flags&FlagCheck != 0 {
        for _, dep := range n.deps {
            s.updateIfNecessary(dep)
            if n.flags&FlagDirty != 0 {
                break // Already dirty, no need to check more
            }
        }
    }

    // Now recompute if truly dirty
    if n.flags&FlagDirty != 0 {
        s.recompute(n)
    }

    n.flags = FlagNone
}
```

### Pending Values Example

```go
func (m *memo[T]) recompute(s *Scheduler) {
    // Compute new value
    newValue := m.fn()

    // Buffer the change
    if m.pendingValue == nil {
        s.pendingNodes = append(s.pendingNodes, m)
    }
    m.pendingValue = &newValue

    // Mark subscribers
    if m.value != newValue || m.pendingValue != nil {
        for _, sub := range m.subs {
            s.dirty.insert(sub)
        }
    }
}

// Later, in stabilize():
func (s *Scheduler) stabilize() {
    // ... process dirty heap ...

    // Commit all changes atomically
    for _, node := range s.pendingNodes {
        if node.pendingValue != nil {
            node.value = *node.pendingValue
            node.pendingValue = nil
        }
    }
}
```

---

## Next Steps

Start with **Phase 1** - implement the heap-based scheduler with glitch prevention. This is the biggest improvement for production readiness and will fundamentally change how the reactive graph executes.

Key files to modify:
- Create `sig/scheduler.go` - new heap-based scheduler
- Modify `sig/memo.go` - add height tracking, pending values
- Modify `sig/signal.go` - integrate with scheduler
- Modify `sig/effect.go` - integrate with scheduler
- Update `sig/batch.go` - use scheduler's batching
