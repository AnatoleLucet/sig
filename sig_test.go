package sig

import (
	"errors"
	"fmt"
	"sync"
)

func ExampleSignal() {
	count := NewSignal(0)
	fmt.Println(count.Read())

	count.Write(10)
	fmt.Println(count.Read())

	// Output:
	// 0
	// 10
}

func ExampleSignal_concurrentRW() {
	var wg sync.WaitGroup
	count := NewSignal(0)

	wg.Go(func() {
		count.Write(count.Read() + 1)
	})

	wg.Wait()
	fmt.Println(count.Read())

	// Output:
	// 1
}

func ExampleSignal_zero() {
	err := NewSignal[error](nil)
	fmt.Println(err.Read())

	err.Write(errors.New("oops"))
	fmt.Println(err.Read())

	err.Write(nil)
	fmt.Println(err.Read())

	// Output:
	// <nil>
	// oops
	// <nil>
}

func ExampleComputed() {
	count := NewSignal(1)
	double := NewComputed(func() int {
		fmt.Println("doubling")
		return count.Read() * 2
	})
	plustwo := NewComputed(func() int {
		fmt.Println("adding")
		return double.Read() + 2
	})
	fmt.Println(count.Read())
	fmt.Println(double.Read())
	fmt.Println(plustwo.Read())

	count.Write(10)
	fmt.Println(count.Read())
	fmt.Println(double.Read())
	fmt.Println(plustwo.Read())

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
	count := NewSignal(1)
	a := NewComputed(func() int {
		fmt.Println("running a")
		return count.Read() * 0 // should never change
	})
	b := NewComputed(func() int {
		fmt.Println("running b")
		return a.Read() + 1
	})
	a.Read()
	b.Read()

	count.Write(10) // should not propagate to b since a did not change

	// Output:
	// running a
	// running b
	// running a
}

func ExampleComputed_disposal() {
	count := NewSignal(1)
	double := NewComputed(func() int {
		fmt.Println("computing")

		NewEffect(func() {
			fmt.Println("effect", count.Read())

			OnCleanup(func() {
				fmt.Println("cleanup", count.Read())
			})
		})

		return count.Read() * 2
	})

	fmt.Println(double.Read())

	count.Write(10)
	fmt.Println(double.Read())

	// Output:
}

func ExampleEffect() {
	count := NewSignal(0)

	fmt.Println(count.Read())

	NewEffect(func() {
		fmt.Println("changed", count.Read())

		OnCleanup(func() {
			fmt.Println("cleanup")
		})
	})

	count.Write(10)
	fmt.Println(count.Read())
	count.Write(20)

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
	count := NewSignal(0)
	double := NewSignal(0)

	NewEffect(func() {
		double.Write(count.Read() * 2)
	})

	NewEffect(func() {
		fmt.Println("changed", double.Read())

		OnCleanup(func() {
			fmt.Println("cleanup")
		})
	})

	count.Write(10)

	// Output:
	// changed 0
	// cleanup
	// changed 20
}

func ExampleEffect_nested() {
	count := NewSignal(0)

	NewEffect(func() {
		count.Read()
		fmt.Println("running")

		NewEffect(func() {
			fmt.Println("running nested")

			OnCleanup(func() {
				fmt.Println("cleanup nested")
			})
		})

		OnCleanup(func() {
			fmt.Println("cleanup")
		})
	})

	count.Write(10)

	// Output:
	// running
	// running nested
	// cleanup nested
	// cleanup
	// running
	// running nested
}

func ExampleEffect_diamond() {
	count := NewSignal(0)
	double := NewComputed(func() int { return count.Read() * 2 })
	quad := NewComputed(func() int { return count.Read() * 4 })

	NewEffect(func() {
		fmt.Println("running", double.Read(), quad.Read())

		OnCleanup(func() {
			fmt.Println("cleanup", double.Read(), quad.Read())
		})
	})

	count.Write(10)

	// Output:
	// running 0 0
	// cleanup 20 40
	// running 20 40
}

func ExampleEffect_diamondNested() {
	count := NewSignal(0)
	double := NewComputed(func() int { return count.Read() * 2 })
	quad := NewComputed(func() int { return count.Read() * 4 })

	NewEffect(func() {
		fmt.Println("running", double.Read(), quad.Read())

		NewEffect(func() {
			fmt.Println("running nested", double.Read(), quad.Read())
			OnCleanup(func() { fmt.Println("cleanup nested", double.Read(), quad.Read()) })
		})

		OnCleanup(func() { fmt.Println("cleanup", double.Read(), quad.Read()) })
	})

	count.Write(10)

	// Output:
	// running 0 0
	// running nested 0 0
	// cleanup nested 20 40
	// cleanup 20 40
	// running 20 40
	// running nested 20 40
}

func ExampleEffect_depsChange() {
	count := NewSignal(0)

	initialized := false
	NewEffect(func() {
		fmt.Println("running")
		if !initialized {
			count.Read()
		}
		initialized = true
	})

	count.Write(1)
	count.Write(2)

	// Output:
	// running
	// running
}

func ExampleEffect_concurrentRW() {
	var wg sync.WaitGroup
	count := NewSignal(0)

	NewEffect(func() {
		fmt.Println(count.Read())

	})

	wg.Go(func() {
		for count.Read() < 5 {
			count.Write(count.Read() + 1)
		}
	})

	wg.Wait()

	// Output:
	// 0
	// 1
	// 2
	// 3
	// 4
	// 5
}

func ExampleEffect_doubleConcurrentRW() {
	var wg sync.WaitGroup
	a := NewSignal(0)
	b := NewSignal(0)

	wg.Go(func() {
		for b.Read() < 5 {
			b.Write(b.Read() + 1)
		}
	})

	wg.Go(func() {
		a.Read()
		a.Write(1)
	})

	NewEffect(func() {
		fmt.Println(a.Read())
	})

	wg.Wait()

	// Output:
	// 0
	// 1
}

func ExampleNewBatch() {
	count := NewSignal(0)

	NewEffect(func() {
		fmt.Println("changed", count.Read())

		OnCleanup(func() {
			fmt.Println("cleanup")
		})
	})

	NewBatch(func() {
		count.Write(10)
		count.Write(20)
		fmt.Println("updated")
	})

	// Output:
	// changed 0
	// updated
	// cleanup
	// changed 20
}

func ExampleNewBatch_double() {
	count := NewSignal(0)
	double := NewSignal(0)

	NewEffect(func() {
		fmt.Println("count", count.Read())

		OnCleanup(func() {
			fmt.Println("count cleanup")
		})
	})

	NewEffect(func() {
		fmt.Println("double", double.Read())

		OnCleanup(func() {
			fmt.Println("double cleanup")
		})
	})

	NewBatch(func() {
		count.Write(10)
		double.Write(count.Read() * 2)
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

func ExampleNewBatch_nested() {
	count := NewSignal(0)

	NewEffect(func() {
		fmt.Println("changed", count.Read())

		OnCleanup(func() {
			fmt.Println("cleanup")
		})
	})

	NewBatch(func() {
		count.Write(10)
		NewBatch(func() {
			count.Write(20)
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
	o := NewOwner()

	o.Run(func() error {
		NewEffect(func() {
			fmt.Println("effect")

			OnCleanup(func() { fmt.Println("cleanup") })
		})

		return nil
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
	o := NewOwner()
	o.OnDispose(func() {
		fmt.Println("parent disposed")
	})

	o.Run(func() error {
		NewOwner().OnDispose(func() {
			fmt.Println("child disposed")
		})

		return nil
	})

	o.Dispose()

	// Output:
	// child disposed
	// parent disposed
}

func ExampleOwner_siblings() {
	o := NewOwner()

	o.Run(func() error {
		OnCleanup(func() {
			fmt.Println("cleanup")
		})

		NewEffect(func() {
			fmt.Println("running first")

			NewEffect(func() {
				fmt.Println("running nested")
				OnCleanup(func() { fmt.Println("cleanup nested") })
			})

			OnCleanup(func() { fmt.Println("cleanup first") })
		})

		NewEffect(func() {
			fmt.Println("running second")
			OnCleanup(func() { fmt.Println("cleanup second") })
		})

		return nil
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
	o := NewOwner()
	o.OnError(func(err any) {
		fmt.Println("cought", err)
	})

	var err *Signal[error]

	o.Run(func() error {
		// should propagate if owner has no error listener
		NewOwner().Run(func() error {
			err = NewSignal[error](nil)

			NewEffect(func() {
				if e := err.Read(); e != nil {
					panic(e)
				}
			})

			return nil
		})

		return nil
	})

	// check if panic in effects are caught
	err.Write(errors.New("oops"))

	// Output:
	// cought oops
}

func ExampleOwner_disposal() {
	o := NewOwner()

	count := NewSignal(0)

	o.Run(func() error {
		NewEffect(func() {
			fmt.Println("effect", count.Read())
		})

		return nil
	})

	count.Write(1)
	o.Dispose()

	// this should not trigger the effect
	count.Write(2)

	// Output:
	// effect 0
	// effect 1
}

func ExampleOwner_effectDisposal() {
	o := NewOwner()

	count := NewSignal(0)

	NewEffect(func() {
		if count.Read() > 0 {
			o.Dispose()
		}
	})

	o.Run(func() error {
		NewEffect(func() {
			fmt.Println("inside", count.Read())
		})

		return nil
	})

	count.Write(1)

	// Output:
	// inside 0
}

func ExampleUntrack() {
	count := NewSignal(0)

	NewEffect(func() {
		c := Untrack(count.Read)
		fmt.Println("effect", c)
	})

	count.Write(10)

	// Output:
	// effect 0
}
