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

func ExampleEffect() {
	count, _ := Signal(0)

	Effect(func() func() {
		count()

		return nil
	})

	count()

	// Output:
}
