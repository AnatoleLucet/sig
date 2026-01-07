package sig

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwner(t *testing.T) {
	t.Run("runs function and disposes", func(t *testing.T) {
		log := []string{}

		o := NewOwner()

		o.Run(func() error {
			NewEffect(func() {
				log = append(log, "effect")

				OnCleanup(func() { log = append(log, "cleanup") })
			})

			return nil
		})

		log = append(log, "ran")
		o.Dispose()
		log = append(log, "disposed")

		assert.Equal(t, []string{
			"effect",
			"ran",
			"cleanup",
			"disposed",
		}, log)
	})

	t.Run("nested owners", func(t *testing.T) {
		log := []string{}

		o := NewOwner()
		o.OnDispose(func() {
			log = append(log, "parent disposed")
		})

		o.Run(func() error {
			NewOwner().OnDispose(func() {
				log = append(log, "child disposed")
			})

			return nil
		})

		o.Dispose()

		assert.Equal(t, []string{
			"child disposed",
			"parent disposed",
		}, log)
	})

	t.Run("sibling effects disposal order", func(t *testing.T) {
		log := []string{}

		o := NewOwner()

		o.Run(func() error {
			OnCleanup(func() {
				log = append(log, "cleanup")
			})

			NewEffect(func() {
				log = append(log, "running first")

				NewEffect(func() {
					log = append(log, "running nested")
					OnCleanup(func() { log = append(log, "cleanup nested") })
				})

				OnCleanup(func() { log = append(log, "cleanup first") })
			})

			NewEffect(func() {
				log = append(log, "running second")
				OnCleanup(func() { log = append(log, "cleanup second") })
			})

			return nil
		})

		log = append(log, "ran")
		o.Dispose()
		log = append(log, "disposed")

		assert.Equal(t, []string{
			"running first",
			"running nested",
			"running second",
			"ran",
			"cleanup second",
			"cleanup nested",
			"cleanup first",
			"cleanup",
			"disposed",
		}, log)
	})

	t.Run("catches panics with OnError", func(t *testing.T) {
		log := []string{}

		o := NewOwner()
		o.OnError(func(err any) {
			log = append(log, fmt.Sprintf("caught %v", err))
		})

		var errSignal *Signal[error]

		o.Run(func() error {
			// should propagate if owner has no error listener
			NewOwner().Run(func() error {
				errSignal = NewSignal[error](nil)

				NewEffect(func() {
					if e := errSignal.Read(); e != nil {
						panic(e)
					}
				})

				return nil
			})

			return nil
		})

		// check if panic in effects are caught
		errSignal.Write(errors.New("oops"))

		assert.Equal(t, []string{
			"caught oops",
		}, log)
	})

	t.Run("disposal prevents effect re-runs", func(t *testing.T) {
		log := []int{}

		o := NewOwner()

		count := NewSignal(0)

		o.Run(func() error {
			NewEffect(func() {
				log = append(log, count.Read())
			})

			return nil
		})

		count.Write(1)
		o.Dispose()

		// this should not trigger the effect
		count.Write(2)

		assert.Equal(t, []int{0, 1}, log)
	})

	t.Run("disposal during effect execution", func(t *testing.T) {
		log := []int{}

		o := NewOwner()

		count := NewSignal(0)

		NewEffect(func() {
			if count.Read() > 0 {
				o.Dispose()
			}
		})

		o.Run(func() error {
			NewEffect(func() {
				log = append(log, count.Read())
			})

			return nil
		})

		count.Write(1)

		assert.Equal(t, []int{0}, log)
	})
}
