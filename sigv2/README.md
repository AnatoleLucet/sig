# sigv2 - Ryan's v2 Reactive System in Go

A complete port of Ryan Solid's v2 async reactive system to Go, featuring full SolidJS-like reactivity with async support, effects, context, and optimistic updates.

## Architecture

### Core Components (12 files, ~2400 lines)

**Infrastructure:**
- `types.go` - Core types, flags, interfaces
- `node.go` - Base reactive node with dependency graph
- `heap.go` - Height-ordered priority queue for topological execution

**Reactive Primitives:**
- `signal.go` - Reactive signals with pending value support
- `computed.go` - Computed values with automatic dependency tracking
- `owner.go` - Ownership scopes, roots, and disposal management

**Scheduling:**
- `scheduler.go` - Queue system, flush mechanism, batch support
- `transition.go` - Async transition tracking

**Advanced Features:**
- `effect.go` - Side effects (render & user types)
- `context_api.go` - Context API for passing values through reactive tree
- `optimistic.go` - Optimistic updates, pending checks
- `api.go` - Public API following sig/ patterns

## Key Design Decisions

✅ **Thread-safe** - Uses `sync.RWMutex` on all nodes
✅ **Goroutine-local context** - Tracks active reactive context per goroutine using `goid`
✅ **Go generics** - Type-safe `Signal[T]`, `Computed[T]` with appropriate constraints
✅ **Factory functions** - Returns getter/setter closures like sig/
✅ **Interface-based dispatch** - `Committable` and `Recomputable` interfaces for type-erased operations
✅ **Async via channels** - Replaces JS Promises with Go channels
✅ **Batch support** - Per-goroutine batch depth tracking
✅ **Goroutine scheduler** - Async flush with infinite loop detection

## API

### Signals
```go
// Create a signal
count, setCount := sigv2.Signal(0)

// Read the value
fmt.Println(count()) // 0

// Update the value
setCount(10)
```

### Computed Values
```go
a, setA := sigv2.Signal(5)
b, setB := sigv2.Signal(3)

// Create computed value - automatically tracks dependencies
sum := sigv2.Computed(func() int {
    return a() + b()
})

fmt.Println(sum()) // 8

setA(10)
// Wait for flush...
fmt.Println(sum()) // 13
```

### Effects
```go
count, setCount := sigv2.Signal(0)

// Create effect that runs when dependencies change
sigv2.Effect(
    func() int { return count() },
    func(value, prev int) sigv2.DisposalFunc {
        fmt.Printf("Count changed from %d to %d\n", prev, value)
        return nil // Optional cleanup function
    },
)
```

### Async Computed
```go
// Create async computed with channel-based result
data, refresh := sigv2.Async(
    func(prev string, refreshing bool) sigv2.AsyncResult[string] {
        return &sigv2.Promise[string]{
            fn: func() (string, error) {
                // Simulate async operation
                time.Sleep(100 * time.Millisecond)
                return "fetched data", nil
            },
        }
    },
    "",
)

// Read the value (throws NotReadyError if not ready)
fmt.Println(data())

// Force refresh
refresh()
```

### Context API
```go
// Create a context
themeCtx := sigv2.CreateContext[string](nil, "theme")

sigv2.Root(func(dispose func()) {
    // Set context value
    sigv2.SetContext(themeCtx, "dark")

    // Get context value
    theme, _ := sigv2.GetContext(themeCtx)
    fmt.Println(theme) // "dark"
})
```

### Batching
```go
count, setCount := sigv2.Signal(0)

// Batch multiple updates - only triggers one flush
sigv2.Batch(func() int {
    setCount(1)
    setCount(2)
    setCount(3)
    return 0
})
```

### Utilities
```go
// Untrack - read without creating dependency
value := sigv2.Untrack(func() int {
    return count() // Won't track as dependency
})

// OnCleanup - register cleanup function
sigv2.OnCleanup(func() {
    fmt.Println("Cleaning up")
})

// Root - create reactive scope
sigv2.Root(func(dispose func()) {
    // Reactive code here
    defer dispose() // Clean up when done
})
```

## Differences from JavaScript Version

### 1. **Channel-based Async**
JavaScript Promises → Go channels:
```go
type AsyncResult[T any] interface {
    Resolve() <-chan Result[T]
}

type Result[T any] struct {
    Value T
    Err   error
}
```

### 2. **Goroutine Context Tracking**
Uses `goid.Get()` for goroutine-local state instead of JavaScript execution context.

### 3. **Asynchronous Updates with WaitForFlush**
Updates are processed asynchronously to avoid deadlocks. Use `WaitForFlush()` when you need to ensure updates are complete:

```go
count, setCount := sigv2.Signal(0)

setCount(10)
sigv2.WaitForFlush() // Wait for update to be processed

fmt.Println(count()) // Now guaranteed to be 10
```

**Why async?** Synchronous updates can cause deadlocks when a signal update triggers a computed update that tries to acquire locks. Async processing avoids this.

**When to use WaitForFlush:**
- In tests when you need to verify the result of an update
- When you need to ensure side effects have completed
- Not needed in normal reactive code (dependencies automatically track)

### 4. **Type Constraints**
- `Signal[T comparable]` - requires equality comparison
- `Computed[T any]` - no constraints
- Effects can track any type

### 5. **Interface-based Dispatch**
Uses `Committable` and `Recomputable` interfaces to handle type-erased operations on generic nodes.

### 6. **Thread-Safe by Default**
All operations are safe to call from multiple goroutines, unlike JavaScript's single-threaded model.

## Performance Characteristics

- **Height-ordered execution** - Ensures dependencies compute before dependents
- **Incremental updates** - Only recomputes affected nodes
- **Batching support** - Coalesces multiple updates
- **Lazy evaluation** - Computed values only update when read
- **Minimal locking** - Read locks for reads, write locks only for updates

## Testing

```bash
go test ./sigv2/... -v
```

All tests passing:
- ✅ Basic signals
- ✅ Computed reactivity
- ✅ Effects
- ✅ Batching
- ✅ Untracking
- ✅ Context API
- ✅ Cleanup

## Thread Safety

All operations are thread-safe:
- Signals can be updated from any goroutine
- Computed values recompute safely
- Effects execute in queue order
- Context is goroutine-local

## Comparison with sig/

| Feature | sig/ | sigv2/ |
|---------|------|--------|
| Basic signals | ✅ | ✅ |
| Computed | ✅ (Memo) | ✅ |
| Effects | ✅ | ✅ (Render & User) |
| Async support | ❌ | ✅ |
| Context API | ❌ | ✅ |
| Transitions | ❌ | ✅ |
| Optimistic updates | ❌ | ✅ |
| Batching | ✅ | ✅ |
| Lines of code | ~430 | ~2400 |

## Future Enhancements

- [ ] Resource API for data fetching
- [ ] Suspense boundaries
- [ ] Stores (nested reactivity)
- [ ] DevTools integration
- [ ] Performance profiling
- [ ] Benchmark suite vs sig/

## Credits

Based on Ryan Solid's v2 async reactive system implementation for JavaScript.
Ported to Go with thread-safety and idiomatic Go patterns.
