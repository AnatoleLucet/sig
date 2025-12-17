//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/AnatoleLucet/sig"
)

func main() {
	doc := js.Global().Get("document")

	count, setCount := sig.Signal(0)

	sig.Effect(func() func() {
		doc.Call("getElementById", "count").Set("textContent", count())
		return nil
	})

	doc.Call("getElementById", "btn").Call("addEventListener", "click",
		js.FuncOf(func(this js.Value, args []js.Value) any {
			setCount(count() + 1)
			return nil
		}))

	<-make(chan struct{})
}
