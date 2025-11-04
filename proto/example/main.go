package main

import (
	"fmt"

	"github.com/AnatoleLucet/sig/proto"
)

func main() {
	fmt.Println("=== Basic Signal and Memo ===")
	basicExample()

	fmt.Println("\n=== Glitch Prevention Demo ===")
	glitchPreventionDemo()

	fmt.Println("\n=== Batching Demo ===")
	batchingDemo()

	fmt.Println("\n=== Diamond Dependency ===")
	diamondDemo()
}

func basicExample() {
	count, setCount := proto.Signal(0)
	doubled := proto.Memo(func() any {
		c := count().(int)
		fmt.Printf("  Computing doubled: %d * 2\n", c)
		return c * 2
	})

	proto.Effect(func() {
		fmt.Printf("  Effect: count=%d, doubled=%d\n", count(), doubled())
	})

	setCount(5)
	setCount(10)
}

func glitchPreventionDemo() {
	// Classic diamond problem that shows glitch prevention
	a, setA := proto.Signal(1)
	b := proto.Memo(func() any {
		result := a().(int) * 2
		fmt.Printf("  Computing b: a * 2 = %d\n", result)
		return result
	})
	c := proto.Memo(func() any {
		aVal := a().(int)
		bVal := b().(int)
		result := aVal + bVal
		fmt.Printf("  Computing c: a + b = %d + %d = %d\n", aVal, bVal, result)
		return result
	})

	proto.Effect(func() {
		fmt.Printf("  Result: a=%d, b=%d, c=%d\n", a(), b(), c())
	})

	fmt.Println("  Updating a to 2...")
	setA(2)
	fmt.Println("  Notice: b computes before c (height-based ordering prevents glitches)")
}

func batchingDemo() {
	firstName, setFirstName := proto.Signal("John")
	lastName, setLastName := proto.Signal("Doe")

	fullName := proto.Memo(func() any {
		full := firstName().(string) + " " + lastName().(string)
		fmt.Printf("  Computing full name: %s\n", full)
		return full
	})

	proto.Effect(func() {
		fmt.Printf("  Full name: %s\n", fullName())
	})

	fmt.Println("  Without batching (two updates):")
	setFirstName("Jane")
	setLastName("Smith")

	fmt.Println("\n  With batching (one update):")
	proto.Batch(func() {
		setFirstName("Alice")
		setLastName("Johnson")
	})
}

func diamondDemo() {
	//     a
	//    / \
	//   b   c
	//    \ /
	//     d

	a, setA := proto.Signal(1)
	b := proto.Memo(func() any {
		result := a().(int) * 2
		fmt.Printf("  b = a * 2 = %d\n", result)
		return result
	})
	c := proto.Memo(func() any {
		result := a().(int) * 3
		fmt.Printf("  c = a * 3 = %d\n", result)
		return result
	})
	d := proto.Memo(func() any {
		bVal := b().(int)
		cVal := c().(int)
		result := bVal + cVal
		fmt.Printf("  d = b + c = %d + %d = %d\n", bVal, cVal, result)
		return result
	})

	proto.Effect(func() {
		fmt.Printf("  Final: a=%d, b=%d, c=%d, d=%d\n", a(), b(), c(), d())
	})

	fmt.Println("  Updating a to 2...")
	setA(2)
	fmt.Println("  Notice: height-based execution: b and c first (height 1), then d (height 2)")
}
