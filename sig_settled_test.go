package sig

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnSettled(t *testing.T) {
	t.Run("runs when flush finishes", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		OnSettled(func() {
			log = append(log, "settled")
		})

		count.Write(10)

		assert.Equal(t, []string{
			"changed 0",
			"cleanup",
			"changed 10",
			"settled",
		}, log)
	})

	t.Run("waits for chained effects", func(t *testing.T) {
		log := []string{}

		a := NewSignal(0)
		b := NewSignal(0)

		NewEffect(func() {
			log = append(log, fmt.Sprintf("A changed %d", a.Read()))

			b.Write(a.Read() * 2)

			OnCleanup(func() {
				log = append(log, "A cleanup")
			})
		})

		NewEffect(func() {
			log = append(log, fmt.Sprintf("B changed %d", b.Read()))

			OnCleanup(func() {
				log = append(log, "B cleanup")
			})
		})

		OnSettled(func() {
			log = append(log, "settled")
		})

		a.Write(10)

		assert.Equal(t, []string{
			"A changed 0",
			"B changed 0",
			"A cleanup",
			"A changed 10",
			"B cleanup",
			"B changed 20",
			"settled",
		}, log)
	})

	t.Run("runs once", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)
		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		OnSettled(func() {
			log = append(log, "settled")
		})

		count.Write(10)
		count.Write(20)

		assert.Equal(t, []string{
			"changed 0",
			"cleanup",
			"changed 10",
			"settled",
			"cleanup",
			"changed 20",
		}, log)
	})

	t.Run("from a goroutine", func(t *testing.T) {
		var wg sync.WaitGroup
		log := []string{}

		count := NewSignal(0)
		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		wg.Go(func() {
			OnSettled(func() {
				log = append(log, "settled")
			})

			count.Write(10)
		})

		wg.Wait()

		assert.Equal(t, []string{
			"changed 0",
			"cleanup",
			"changed 10",
			"settled",
		}, log)
	})
}

func TestOnUserSettled(t *testing.T) {
	t.Run("runs after user effects", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		OnUserSettled(func() {
			log = append(log, "settled")
		})

		count.Write(10)

		assert.Equal(t, []string{
			"changed 0",
			"cleanup",
			"changed 10",
			"settled",
		}, log)
	})

	t.Run("does not wait for chained effects", func(t *testing.T) {
		log := []string{}

		a := NewSignal(0)
		b := NewSignal(0)

		NewEffect(func() {
			log = append(log, fmt.Sprintf("A changed %d", a.Read()))

			b.Write(a.Read() * 2)

			OnCleanup(func() {
				log = append(log, "A cleanup")
			})
		})

		NewEffect(func() {
			log = append(log, fmt.Sprintf("B changed %d", b.Read()))

			OnCleanup(func() {
				log = append(log, "B cleanup")
			})
		})

		OnUserSettled(func() {
			log = append(log, "settled")
		})

		a.Write(10)

		assert.Equal(t, []string{
			"A changed 0",
			"B changed 0",
			"A cleanup",
			"A changed 10",
			"settled",
			"B cleanup",
			"B changed 20",
		}, log)
	})
}

func TestOnRenderSettled(t *testing.T) {
	t.Run("runs after render effects", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		NewRenderEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		OnRenderSettled(func() {
			log = append(log, "settled")
		})

		count.Write(10)

		assert.Equal(t, []string{
			"changed 0",
			"cleanup",
			"changed 10",
			"settled",
		}, log)
	})

	t.Run("does not wait for user effects", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)
		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		OnRenderSettled(func() {
			log = append(log, "settled")
		})

		count.Write(10)

		assert.Equal(t, []string{
			"changed 0",
			"settled",
			"cleanup",
			"changed 10",
		}, log)
	})

	t.Run("does not wait for chained effects", func(t *testing.T) {
		log := []string{}

		a := NewSignal(0)
		b := NewSignal(0)

		NewRenderEffect(func() {
			log = append(log, fmt.Sprintf("A changed %d", a.Read()))
			b.Write(a.Read() * 2)

			OnCleanup(func() {
				log = append(log, "A cleanup")
			})
		})

		NewRenderEffect(func() {
			log = append(log, fmt.Sprintf("B changed %d", b.Read()))

			OnCleanup(func() {
				log = append(log, "B cleanup")
			})
		})

		OnRenderSettled(func() {
			log = append(log, "settled")
		})

		a.Write(10)

		assert.Equal(t, []string{
			"A changed 0",
			"B changed 0",
			"A cleanup",
			"A changed 10",
			"settled",
			"B cleanup",
			"B changed 20",
		}, log)
	})
}
