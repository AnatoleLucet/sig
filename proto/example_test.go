package proto_test

import (
	"fmt"
	"testing"

	"github.com/AnatoleLucet/sig/proto"
)

func TestBasicSignalAndMemo(t *testing.T) {
	count, setCount := proto.Signal(0)
	doubled := proto.Memo(func() any {
		return count().(int) * 2
	})

	if doubled().(int) != 0 {
		t.Errorf("Expected 0, got %d", doubled())
	}

	setCount(5)
	if doubled().(int) != 10 {
		t.Errorf("Expected 10, got %d", doubled())
	}
}

func TestGlitchPrevention(t *testing.T) {
	// This test demonstrates glitch prevention
	a, setA := proto.Signal(1)
	b := proto.Memo(func() any { return a().(int) * 2 })
	c := proto.Memo(func() any { return a().(int) + b().(int) })

	// Initial: a=1, b=2, c=3
	if c().(int) != 3 {
		t.Errorf("Expected c=3, got %d", c())
	}

	// Update a to 2
	// Without glitch prevention, c might see a=2, b=2(old) → c=4 (WRONG!)
	// With glitch prevention, b updates first → b=4, then c sees a=2, b=4 → c=6 (CORRECT!)
	setA(2)

	if b().(int) != 4 {
		t.Errorf("Expected b=4, got %d", b())
	}
	if c().(int) != 6 {
		t.Errorf("Expected c=6, got %d (glitch detected!)", c())
	}
}

func TestEffect(t *testing.T) {
	count, setCount := proto.Signal(0)

	callCount := 0
	var lastValue int

	proto.Effect(func() {
		callCount++
		lastValue = count().(int)
	})

	// Effect runs immediately
	if callCount != 1 || lastValue != 0 {
		t.Errorf("Expected 1 call with value 0, got %d calls with value %d", callCount, lastValue)
	}

	setCount(5)
	if callCount != 2 || lastValue != 5 {
		t.Errorf("Expected 2 calls with value 5, got %d calls with value %d", callCount, lastValue)
	}
}

func TestBatching(t *testing.T) {
	a, setA := proto.Signal(1)
	b, setB := proto.Signal(2)

	callCount := 0
	sum := proto.Memo(func() any {
		callCount++
		return a().(int) + b().(int)
	})

	// Initial computation
	if sum().(int) != 3 {
		t.Errorf("Expected 3, got %d", sum())
	}
	callCount = 0 // Reset

	// Without batching, this would trigger 2 recomputations
	// With batching, only 1 recomputation
	proto.Batch(func() {
		setA(10)
		setB(20)
	})

	if sum().(int) != 30 {
		t.Errorf("Expected 30, got %d", sum())
	}
	if callCount != 1 {
		t.Errorf("Expected 1 recomputation with batching, got %d", callCount)
	}
}

func TestDiamondDependency(t *testing.T) {
	// Classic diamond dependency graph:
	//     a
	//    / \
	//   b   c
	//    \ /
	//     d

	a, setA := proto.Signal(1)
	b := proto.Memo(func() any { return a().(int) * 2 })
	c := proto.Memo(func() any { return a().(int) * 3 })
	d := proto.Memo(func() any { return b().(int) + c().(int) })

	// Initial: a=1, b=2, c=3, d=5
	if d().(int) != 5 {
		t.Errorf("Expected d=5, got %d", d())
	}

	// Update a
	// Without glitch prevention, d might see inconsistent intermediate states
	// With glitch prevention: a→b,c→d (topological order)
	setA(2)

	// Expected: a=2, b=4, c=6, d=10
	if b().(int) != 4 {
		t.Errorf("Expected b=4, got %d", b())
	}
	if c().(int) != 6 {
		t.Errorf("Expected c=6, got %d", c())
	}
	if d().(int) != 10 {
		t.Errorf("Expected d=10, got %d (glitch in diamond dependency!)", d())
	}
}

// Example demonstrates basic usage
func ExampleSignal() {
	count, setCount := proto.Signal(0)

	proto.Effect(func() {
		fmt.Printf("Count: %d\n", count())
	})

	setCount(1)
	setCount(2)

	// Output:
	// Count: 0
	// Count: 1
	// Count: 2
}

// Example demonstrates glitch prevention
func ExampleMemo_glitchPrevention() {
	a, setA := proto.Signal(1)
	b := proto.Memo(func() any { return a().(int) * 2 })
	c := proto.Memo(func() any { return a().(int) + b().(int) })

	proto.Effect(func() {
		fmt.Printf("a=%d, b=%d, c=%d\n", a(), b(), c())
	})

	setA(2)

	// Output:
	// a=1, b=2, c=3
	// a=2, b=4, c=6
}

// Example demonstrates batching
func ExampleBatch() {
	a, setA := proto.Signal(1)
	b, setB := proto.Signal(2)

	sum := proto.Memo(func() any {
		return a().(int) + b().(int)
	})

	proto.Effect(func() {
		fmt.Printf("Sum: %d\n", sum())
	})

	proto.Batch(func() {
		setA(10)
		setB(20)
	})

	// Output:
	// Sum: 3
	// Sum: 30
}
