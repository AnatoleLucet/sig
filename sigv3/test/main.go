package main

import (
	"fmt"

	"github.com/AnatoleLucet/sig/sigv3"
)

func main() {
	runtime := sig.GetRuntime()
	defer runtime.Flush()

	count, setCount := runtime.Signal(0)
	fmt.Println("count:", count())

	runtime.Effect(func() {
		fmt.Println("doubled:", count().(int)*2)
	})

	setCount(10)
	fmt.Println("count:", count())
}
