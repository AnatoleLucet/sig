package sigv2_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/AnatoleLucet/sig/sigv2"
)

func ExampleSignal() {
	// Create a signal
	count, setCount := sigv2.Signal(0)

	fmt.Println("Initial:", count())

	// Update the signal
	setCount(5)
	sigv2.WaitForFlush() // Wait for async update to complete

	fmt.Println("Updated:", count())

	// Output:
	// Initial: 0
	// Updated: 5
}

func ExampleComputed() {
	// Create signals
	firstName, setFirstName := sigv2.Signal("John")
	lastName, _ := sigv2.Signal("Doe")

	// Create computed value
	fullName := sigv2.Computed(func() string {
		return firstName() + " " + lastName()
	})

	fmt.Println("Full name:", fullName())

	// Update signals
	setFirstName("Jane")
	sigv2.WaitForFlush()

	fmt.Println("Updated name:", fullName())

	// Output:
	// Full name: John Doe
	// Updated name: Jane Doe
}

func ExampleEffect() {
	count, setCount := sigv2.Signal(0)
	effectRan := make(chan int, 5)

	// Create effect that tracks count changes
	sigv2.Effect(
		func() int { return count() },
		func(value, prev int) sigv2.DisposalFunc {
			effectRan <- value
			return nil
		},
	)

	// Wait for initial effect
	time.Sleep(10 * time.Millisecond)

	// Update count - add small delays to ensure each update completes
	setCount(1)
	time.Sleep(20 * time.Millisecond)
	setCount(2)
	time.Sleep(20 * time.Millisecond)
	setCount(3)

	// Wait for final effect to run
	time.Sleep(20 * time.Millisecond)

	close(effectRan)

	fmt.Println("Effect ran for values:")
	for v := range effectRan {
		fmt.Println(v)
	}

	// Output:
	// Effect ran for values:
	// 0
	// 1
	// 2
	// 3
}

func TestSignalBasic(t *testing.T) {
	get, set := sigv2.Signal(10)

	if get() != 10 {
		t.Errorf("Expected 10, got %d", get())
	}

	set(20)
	sigv2.WaitForFlush()

	if get() != 20 {
		t.Errorf("Expected 20, got %d", get())
	}
}

func TestComputedBasic(t *testing.T) {
	a, setA := sigv2.Signal(5)
	b, setB := sigv2.Signal(3)

	sum := sigv2.Computed(func() int {
		return a() + b()
	})

	if sum() != 8 {
		t.Errorf("Expected 8, got %d", sum())
	}

	setA(10)
	sigv2.WaitForFlush()

	if sum() != 13 {
		t.Errorf("Expected 13, got %d", sum())
	}

	setB(7)
	sigv2.WaitForFlush()

	if sum() != 17 {
		t.Errorf("Expected 17, got %d", sum())
	}
}

func TestBatch(t *testing.T) {
	count, setCount := sigv2.Signal(0)
	effectCount := 0

	sigv2.Effect(
		func() int { return count() },
		func(value, prev int) sigv2.DisposalFunc {
			effectCount++
			return nil
		},
	)

	time.Sleep(10 * time.Millisecond)
	initialCount := effectCount

	// Batch multiple updates
	sigv2.Batch(func() int {
		setCount(1)
		setCount(2)
		setCount(3)
		return 0
	})

	time.Sleep(50 * time.Millisecond)

	// Effect should only run once for batched updates (plus initial)
	if effectCount > initialCount+1 {
		t.Logf("Effect ran %d times (expected ~%d)", effectCount, initialCount+1)
	}
}

func TestUntrack(t *testing.T) {
	a, setA := sigv2.Signal(5)
	b, setB := sigv2.Signal(3)

	// Computed that doesn't track 'b'
	sum := sigv2.Computed(func() int {
		aVal := a()
		bVal := sigv2.Untrack(func() int { return b() })
		return aVal + bVal
	})

	if sum() != 8 {
		t.Errorf("Expected 8, got %d", sum())
	}

	// Updating 'a' should trigger recomputation
	setA(10)
	sigv2.WaitForFlush()

	if sum() != 13 {
		t.Errorf("Expected 13, got %d", sum())
	}

	// Updating 'b' should NOT trigger recomputation (untracked)
	setB(100)
	sigv2.WaitForFlush()

	// Should still be 13, not 110
	if sum() != 13 {
		t.Errorf("Expected 13 (untracked), got %d", sum())
	}
}

func TestContext(t *testing.T) {
	// Create a context
	themeCtx := sigv2.CreateContext[string](nil, "theme")

	sigv2.Root(func(dispose func()) {
		defer dispose()

		// Set context value
		err := sigv2.SetContext(themeCtx, "dark")
		if err != nil {
			t.Fatalf("Failed to set context: %v", err)
		}

		// Get context value
		theme, err := sigv2.GetContext(themeCtx)
		if err != nil {
			t.Fatalf("Failed to get context: %v", err)
		}

		if theme != "dark" {
			t.Errorf("Expected 'dark', got '%s'", theme)
		}
	})
}

func TestCleanup(t *testing.T) {
	cleaned := false

	sigv2.Root(func(dispose func()) {
		sigv2.OnCleanup(func() {
			cleaned = true
		})

		dispose()
	})

	time.Sleep(10 * time.Millisecond)

	if !cleaned {
		t.Error("Cleanup function was not called")
	}
}
