package main

import (
	"fmt"

	"github.com/AnatoleLucet/sig/old"
)

func main() {
	runtime := sig.GetRuntime()
	defer runtime.Flush()

	runtime.Effect(func() func() {
		count, _ := runtime.Signal(0)
		fmt.Println("count:", count())

		runtime.Effect(func() func() {
			fmt.Println("doubled:", count().(int)*2)

			return func() {
				fmt.Println("cleanup doubled effect")
			}
		})
		runtime.Flush()

		// setCount(10)
		// fmt.Println("count:", count())
		// runtime.Flush()

		return nil
	})
}
