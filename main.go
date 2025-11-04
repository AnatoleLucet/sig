package main

import (
	"fmt"
	"time"

	"github.com/AnatoleLucet/sig/sig"
)

func main() {
	o1 := sig.Owner()

	o1.Run(func() {
		a, setA := sig.Signal(1)
		b, setB := sig.Signal(2)

		sum := sig.Memo(func() int {
			result := a() + b()
			fmt.Println("  [MEMO] Computing sum:", result)
			return result
		})

		sig.Effect(func() {
			fmt.Println("  [EFFECT] Sum is:", sum())
		})

		fmt.Println("\nUpdating both a and b in a batch...")
		sig.Batch(func() {
			setA(10)
			setB(20)
		})

		fmt.Println("\nExpected: sum computes once (30)")
		fmt.Println("Current: sum might compute twice if not properly deduplicated")
	})

	time.Sleep(1 * time.Second)
}
