package sig

import (
	"fmt"
)

func ExampleSignal() {
	count, setCount := Signal(0)
	fmt.Println(count())

	setCount(10)
	fmt.Println(count())

	// Output:
	// 0
	// 10
}

func ExampleComputed() {
	count, setCount := Signal(1)
	double := Computed(func() int {
		fmt.Println("doubling")
		return count() * 2
	})
	fmt.Println(count())
	fmt.Println(double())

	setCount(10)
	fmt.Println(count())
	fmt.Println(double())

	// Output:
	// doubling
	// 1
	// 2
	// doubling
	// 10
	// 20
}

// func ExampleEffect() {
// 	count, _ := Signal(0)
//
// 	Effect(func() func() {
// 		count()
//
// 		return nil
// 	})
//
// 	count()
//
// 	// Output:
// }
