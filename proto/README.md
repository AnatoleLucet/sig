# Proto - Production-Ready Reactive System

This is a prototype implementation of a production-ready reactive system for Go, inspired by modern reactive frameworks.

## Key Features

✅ **Glitch Prevention**: Height-based topological sorting ensures consistent state
✅ **Optimized Recomputation**: Dirty/Check flags minimize unnecessary work
✅ **Atomic Updates**: Pending value system commits all changes together
✅ **Batching**: Queue updates and process them efficiently
✅ **Thread-Safe**: Proper locking for concurrent access

## Architecture

- **Scheduler**: Manages the reactive graph execution with a min-heap
- **Signal**: Primitive reactive values (height 0)
- **Memo**: Computed values that cache results (height 1+)
- **Effect**: Side-effects that run when dependencies change
- **Heap**: Priority queue organized by dependency height

## Usage

```go
package main

import (
    "fmt"
    "github.com/AnatoleLucet/sig/proto"
)

func main() {
    // Create signals
    count, setCount := proto.Signal(0)
    multiplier, setMultiplier := proto.Signal(2)

    // Create computed value
    doubled := proto.Memo(func() any {
        return count().(int) * multiplier().(int)
    })

    // Create effect
    proto.Effect(func() {
        fmt.Printf("Count: %d, Doubled: %d\n", count(), doubled())
    })
    // Output: Count: 0, Doubled: 0

    // Update values
    setCount(5)
    // Output: Count: 5, Doubled: 10

    setMultiplier(3)
    // Output: Count: 5, Doubled: 15

    // Batch multiple updates
    proto.Batch(func() {
        setCount(10)
        setMultiplier(4)
    })
    // Output: Count: 10, Doubled: 40 (runs once, not twice)
}
```

## Example: Glitch Prevention

```go
// Without glitch prevention (old system):
// a=1, b=2, c=3
// Set a=2
// c might see: a=2, b=2(old) → c=4 (WRONG!)

// With glitch prevention (this system):
a, setA := proto.Signal(1)
b := proto.Memo(func() any { return a().(int) * 2 })
c := proto.Memo(func() any { return a().(int) + b().(int) })

proto.Effect(func() {
    fmt.Printf("a=%d, b=%d, c=%d\n", a(), b(), c())
})
// Output: a=1, b=2, c=3

setA(2)
// Output: a=2, b=4, c=6 (CORRECT!)
// b always updates before c because height(b)=1, height(c)=2
```

## How It Works

### 1. Height Tracking
Each node has a height based on its dependency depth:
- Signals: height 0
- Memos: max(dependency heights) + 1
- Effects: dynamic based on dependencies

### 2. Heap-Based Scheduling
The scheduler processes nodes from lowest to highest height, ensuring dependencies run before dependents.

### 3. Pending Values
Changes are buffered and committed atomically:
1. Signal changes → mark subscribers dirty
2. Process dirty heap (recompute memos/effects)
3. Commit all pending values together
4. Increment clock

### 4. Dirty/Check Optimization
- **Dirty**: Node definitely needs recomputation
- **Check**: Node might need recomputation (check dependencies first)

This avoids unnecessary work when a dependency changes but produces the same value.

## Differences from sig/

| Feature | proto/ | sig/ |
|---------|--------|------|
| Glitch Prevention | ✅ Yes | ❌ No |
| Scheduling | Heap-based | Immediate/Queue |
| Pending Values | ✅ Yes | ❌ No |
| Optimization | Dirty/Check flags | Simple dirty bool |
| Complexity | Higher | Lower |

## Limitations

This is a prototype. Not yet implemented:
- Ownership/disposal system
- Async support (Go-style)
- Full thread safety audit
- Performance optimizations (object pooling, etc.)
- Comprehensive error handling

See `../REACTIVE_COMPARISON.md` for the full roadmap.
