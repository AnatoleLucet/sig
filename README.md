<h1 align="center"><code>sig</code></h1>

<p align="center">Reactive signals in Go</p>

```go
count := sig.NewSignal(0)

sig.NewEffect(func() {
    fmt.Println("changed", count.Read())
})

count.Write(10)
```

## Features

- Signals, effects, computed values (memos), batching, untrack, and owners
- Automatic dependency tracking
- Per-goroutine runtime isolation
- Height-based priority scheduling
- Topological ordering
- Infinite loop detection
- Staleness detection
- Zero dependency

Coming soon:

- Contexts
- Async computed values

## Introduction

`sig` is based on the very latest from the SolidJS team ([sou](https://github.com/solidjs/signals)-[rc](https://x.com/RyanCarniato/status/1986922658232156382?s=20)-[e](https://x.com/RyanCarniato/status/1991922576541823275?s=20)-[s](https://github.com/milomg/r3)). It aims to be a fully fledged signal-based reactive model with async first support, that can be embedded anywhere.

> Note: `sig` is purpose built for framework authors to add reactivity in their tool. For a more user-friendly and SolidJS-like API, see [loom/signals](https://github.com/AnatoleLucet/loom/tree/main/core/signals).

## TODOs

<details>
<summary>☑️ signals</summary>

```go
count := sig.NewSignal(0)
fmt.Println(count.Read())

count.Write(10)
fmt.Println(count.Read())

// Output
// 0
// 10
```

</details>

<details>
<summary>☑️ computed</summary>

```go
count := sig.NewSignal(1)
double := sig.NewComputed(func() int {
    fmt.Println("doubling")
    return count.Read()*2
})
fmt.Println(count.Read())
fmt.Println(double.Read())

count.Write(10)
fmt.Println(count.Read())
fmt.Println(double.Read())

// Output:
// doubling
// 1
// 2
// doubling
// 10
// 20
```

</details>

<details>
<summary>☑️ effects</summary>

```go
count := sig.NewSignal(1)
fmt.Println(count.Read())

sig.NewEffect(func() {
    fmt.Println(count.Read()*2)
})

count.Write(10)
fmt.Println(count.Read())

// Output:
// 1
// 2
// 20 -- note that effects run immediately on setCount(). this is different than Solid's reactive system (see Batch() for alternatives)
// 10
```

</details>

<details>
<summary>☑️ batch</summary>

```go
count := sig.NewSignal(1)
fmt.Println(count.Read())

sig.NewEffect(func() {
    fmt.Println(count.Read()*2)
})

sig.NewBatch(func () {
    count.Write(10)
    fmt.Println(count.Read())
})

// Output:
// 1
// 2
// 10
// 20 -- now with batch, effects are defered to the end of the batch, so 10 is logged before 20.
//       batch can also be used to update a state multiple times while making sure its effects are only run once.
```

</details>

<details>
<summary>☑️ owner</summary>

```go
// mainly used by framework authors to "own" a reactive context and dispose it when appropriate
owner := sig.NewOwner()
owner.OnError(func (err any) {
    fmt.Println("recovered:", err)
})

owner.Run(func() {
    count := sig.NewSignal(1)
    fmt.Println(count.Read())

    sig.NewEffect(func() {
        fmt.Println(count.Read()*2)
        sig.OnCleanup(func() {
            fmt.Println("disposed")
        })
    })

    count.Write(10)
    fmt.Println(count.Read())
})

owner.Dispose()

// Output:
// doubling
// 1
// 2
// disposed
// 20
// 10
// disposed
```

</details>

<details>
<summary>⬜ async computed</summary>

```go
userID := sig.NewSignal(0)
user := sig.NewAsyncComputed(func() (User, error) { // func is called in a goroutine
    return getUser(userID.Read())
})

sig.NewEffect(func() {
    if sig.IsPending(user) { // uses the panic logic to know if the computed node has resolved yet or not
        fmt.Println("loading...")
        return nil
    }

    // if we're in a reactive scope and user has not resolved yet, this will panic and be recovered to tell the node one of its dependencies is not ready.
    // else it returns an error to avoid panics in a scope not owned by the reactive system.
    u, err := user.Read()
    if err {
        fmt.Println("error:", err)
        return nil
    }

    fmt.Println("user:", u.Name)
    return nil
})

// Output:
// loading...
// user: Bob
```

</details>

<details>
<summary>⬜ context</summary>

```go
ctx := sig.NewContext("light") // default value

owner := sig.NewOwner()
owner.Run(func() {
    ctx.Set("dark")

    sig.NewOwner().Run(func() {
        theme := ctx.Get()
        fmt.Println(theme)
    })
})

theme := ctx.Get() // returns default value
fmt.Println(theme)

// Output:
// dark
// light
```

</details>

<details>
<summary>☑️ untrack</summary>

```go
count := sig.NewSignal(1)
other := sig.NewSignal(10)

sig.NewEffect(func() {
    fmt.Println(count.Read(), sig.Untrack(other.Read))
})

count.Write(2)
other.Write(20)

// Output:
// 1, 10
// 2, 10 -- stops here and no effect is triggered for the setOther(20)
```

</details>

## FAQ

#### Differences with SolidJS's reactivity model

TODO: instant flush and batching, multi-threading for async computed, no need for async effects because you can just `go fn()` wherever to go async

## Credits

- Ryan Carniato, Milo Mighdoll, and everyone else involved in the JS community for pushing the limits of what's possible with reactive systems.
