//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/AnatoleLucet/sig"
)

func main() {
	doc := js.Global().Get("document")

	count := sig.NewSignal(0)

	sig.NewEffect(func() {
		doc.Call("getElementById", "count").Set("textContent", count.Read())
	})

	doc.Call("getElementById", "btn").Call("addEventListener", "click",
		js.FuncOf(func(this js.Value, args []js.Value) any {
			count.Write(count.Read() + 1)
			return nil
		}))

	<-make(chan struct{})
}
