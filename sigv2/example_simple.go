// +build ignore

package main

import (
	"fmt"

	"github.com/AnatoleLucet/sig/sigv2"
)

func main() {
	// Create signals
	firstName, setFirstName := sigv2.Signal("John")
	lastName, setLastName := sigv2.Signal("Doe")

	// Create computed value - automatically tracks dependencies!
	fullName := sigv2.Computed(func() string {
		return firstName() + " " + lastName()
	})

	// Create effect that runs when dependencies change
	sigv2.Effect(
		func() string { return fullName() },
		func(name, prev string) sigv2.DisposalFunc {
			fmt.Printf("Name changed from '%s' to '%s'\n", prev, name)
			return nil
		},
	)

	fmt.Println("Initial:", fullName())

	// Update signals
	setFirstName("Jane")
	sigv2.WaitForFlush()

	setLastName("Smith")
	sigv2.WaitForFlush()

	fmt.Println("Final:", fullName())
}
