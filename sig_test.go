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

func ExampleComputed_check() {
	count, setCount := Signal(1)
	a := Computed(func() int {
		fmt.Println("running a")
		return count() * 0 // should never change
	})
	b := Computed(func() int {
		fmt.Println("running b")
		return a() + 1
	})
	a()
	b()

	setCount(10) // should not propagate to b since a did not change

	// Output:
	// running a
	// running b
	// running a
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
	setCount(20)

	// Output:
	// 0
	// changed 0
	// cleanup
	// changed 10
	// 10
	// cleanup
	// changed 20
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
		count()
		fmt.Println("running")

		Effect(func() func() {
			fmt.Println("running nested")

			return func() {
				fmt.Println("cleanup nested")
			}
		})

		return func() {
			fmt.Println("cleanup")
		}
	})

	setCount(10)

	// Output:
	// running
	// running nested
	// cleanup nested
	// cleanup
	// running
	// running nested
}

func ExampleEffect_diamond() {
	count, setCount := Signal(0)
	double := Computed(func() int { return count() * 2 })
	quad := Computed(func() int { return count() * 4 })

	Effect(func() func() {
		fmt.Println("running", double(), quad())

		return func() {
			fmt.Println("cleanup", double(), quad())
		}
	})

	setCount(10)

	// Output:
	// running 0 0
	// cleanup 20 40
	// running 20 40
}

func ExampleEffect_diamondNested() {
	count, setCount := Signal(0)
	double := Computed(func() int { return count() * 2 })
	quad := Computed(func() int { return count() * 4 })

	Effect(func() func() {
		fmt.Println("running", double(), quad())

		Effect(func() func() {
			fmt.Println("running nested", double(), quad())
			return func() { fmt.Println("cleanup nested", double(), quad()) }
		})

		return func() { fmt.Println("cleanup", double(), quad()) }
	})

	setCount(10)

	// Output:
	// running 0 0
	// running nested 0 0
	// cleanup nested 20 40
	// cleanup 20 40
	// running 20 40
	// running nested 20 40
}

func ExampleEffect_depsChange() {
	count1, setCount1 := Signal(0)

	initialized := false
	Effect(func() func() {
		fmt.Println("running")
		if !initialized {
			count1()
		}
		initialized = true

		return nil
	})

	setCount1(1)
	setCount1(2)

	// Output:
	// running
	// running
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

func ExampleOwner() {
	o := Owner()

	o.Run(func() {
		Effect(func() func() {
			fmt.Println("effect")

			return func() { fmt.Println("cleanup") }
		})
	})

	fmt.Println("ran")
	o.Dispose()
	fmt.Println("disposed")

	// Output:
	// effect
	// ran
	// cleanup
	// disposed
}

func ExampleOwner_nested() {
	o := Owner()
	o.OnDispose(func() {
		fmt.Println("parent disposed")
	})

	o.Run(func() {
		Owner().OnDispose(func() {
			fmt.Println("child disposed")
		})
	})

	o.Dispose()

	// Output:
	// child disposed
	// parent disposed
}

func ExampleOwner_siblings() {
	o := Owner()

	o.Run(func() {
		OnCleanup(func() {
			fmt.Println("cleanup")
		})

		Effect(func() func() {
			fmt.Println("running first")

			Effect(func() func() {
				fmt.Println("running nested")
				return func() { fmt.Println("cleanup nested") }
			})

			return func() { fmt.Println("cleanup first") }
		})

		Effect(func() func() {
			fmt.Println("running second")
			return func() { fmt.Println("cleanup second") }
		})
	})

	fmt.Println("ran")
	o.Dispose()
	fmt.Println("disposed")

	// Output:
	// running first
	// running nested
	// running second
	// ran
	// cleanup second
	// cleanup nested
	// cleanup first
	// cleanup
	// disposed
}

func ExampleOwner_onError() {
	o := Owner()
	o.OnError(func(err any) {
		fmt.Println("cought", err)
	})

	o.Run(func() {
		// should propagate if owner has no error listener
		Owner().Run(func() { panic("oops") })
	})

	// Output:
	// cought oops
}

func ExampleUntrack() {
	count, setCount := Signal(0)

	Effect(func() func() {
		c := Untrack(count)
		fmt.Println("effect", c)
		return nil
	})

	setCount(10)

	// Output:
	// effect 0
}
