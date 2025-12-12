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
	plustwo := Computed(func() int {
		fmt.Println("adding")
		return double() + 2
	})
	fmt.Println(count())
	fmt.Println(double())
	fmt.Println(plustwo())

	setCount(10)
	fmt.Println(count())
	fmt.Println(double())
	fmt.Println(plustwo())

	// Output:
	// doubling
	// adding
	// 1
	// 2
	// 4
	// doubling
	// adding
	// 10
	// 20
	// 22
}

func ExampleEffect() {
	count, setCount := Signal(0)

	fmt.Println(count())

	Effect(func() func() {
		fmt.Println("changed", count())

		return func() {
			fmt.Println("cleanup")
		}
	})

	setCount(10)
	fmt.Println(count())

	// Output:
	// 0
	// changed 0
	// cleanup
	// changed 10
	// 10
}

func ExampleEffect_double() {
	count, setCount := Signal(0)
	double, setDouble := Signal(0)

	Effect(func() func() {
		setDouble(count() * 2)

		return nil
	})

	Effect(func() func() {
		fmt.Println("changed", double())

		return func() {
			fmt.Println("cleanup")
		}
	})

	setCount(10)

	// Output:
	// changed 0
	// cleanup
	// changed 20
}

func ExampleEffect_nested() {
	count, setCount := Signal(0)

	Effect(func() func() {
		fmt.Println("count", count())

		Effect(func() func() {
			fmt.Println("nested count", count())

			return func() {
				fmt.Println("nested cleanup")
			}
		})

		return func() {
			fmt.Println("cleanup")
		}
	})

	setCount(10)

	// Output:
	// count 0
	// nested count 0
	// nested cleanup
	// nested count 10
	// cleanup
	// count 10
}

func ExampleBatch() {
	count, setCount := Signal(0)

	Effect(func() func() {
		fmt.Println("changed", count())

		return func() {
			fmt.Println("cleanup")
		}
	})

	Batch(func() {
		setCount(10)
		setCount(20)
		fmt.Println("updated")
	})

	// Output:
	// changed 0
	// updated
	// cleanup
	// changed 20
}

func ExampleBatch_double() {
	count, setCount := Signal(0)
	double, setDouble := Signal(0)

	Effect(func() func() {
		fmt.Println("count", count())

		return func() {
			fmt.Println("count cleanup")
		}
	})

	Effect(func() func() {
		fmt.Println("double", double())

		return func() {
			fmt.Println("double cleanup")
		}
	})

	Batch(func() {
		setCount(10)
		setDouble(count() * 2)
		fmt.Println("updated")
	})

	// Output:
	// count 0
	// double 0
	// updated
	// count cleanup
	// count 10
	// double cleanup
	// double 20
}

func ExampleBatch_nested() {
	count, setCount := Signal(0)

	Effect(func() func() {
		fmt.Println("changed", count())

		return func() {
			fmt.Println("cleanup")
		}
	})

	Batch(func() {
		setCount(10)
		Batch(func() {
			setCount(20)
		})
		fmt.Println("updated")
	})

	// Output:
	// changed 0
	// updated
	// cleanup
	// changed 20
}
