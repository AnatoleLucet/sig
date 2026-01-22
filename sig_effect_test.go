package sig

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEffect(t *testing.T) {
	t.Run("runs on signal change with cleanup", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)
		log = append(log, fmt.Sprintf("%d", count.Read()))

		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		count.Write(10)
		log = append(log, fmt.Sprintf("%d", count.Read()))
		count.Write(20)

		assert.Equal(t, []string{
			"0",
			"changed 0",
			"cleanup",
			"changed 10",
			"10",
			"cleanup",
			"changed 20",
		}, log)
	})

	t.Run("writes to another signal", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)
		double := NewSignal(0)

		NewEffect(func() {
			double.Write(count.Read() * 2)
		})

		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", double.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		count.Write(10)

		assert.Equal(t, []string{
			"changed 0",
			"cleanup",
			"changed 20",
		}, log)
	})

	t.Run("nested effects", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		NewEffect(func() {
			count.Read()
			log = append(log, "running")

			NewEffect(func() {
				log = append(log, "running nested")

				OnCleanup(func() {
					log = append(log, "cleanup nested")
				})
			})

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		count.Write(10)

		assert.Equal(t, []string{
			"running",
			"running nested",
			"cleanup nested",
			"cleanup",
			"running",
			"running nested",
		}, log)
	})

	t.Run("diamond dependency", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)
		double := NewComputed(func() int { return count.Read() * 2 })
		quad := NewComputed(func() int { return count.Read() * 4 })

		NewEffect(func() {
			log = append(log, fmt.Sprintf("running %d %d", double.Read(), quad.Read()))

			OnCleanup(func() {
				log = append(log, fmt.Sprintf("cleanup %d %d", double.Read(), quad.Read()))
			})
		})

		count.Write(10)

		assert.Equal(t, []string{
			"running 0 0",
			"cleanup 20 40",
			"running 20 40",
		}, log)
	})

	t.Run("diamond dependency nested", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)
		double := NewComputed(func() int { return count.Read() * 2 })
		quad := NewComputed(func() int { return count.Read() * 4 })

		NewEffect(func() {
			log = append(log, fmt.Sprintf("running %d %d", double.Read(), quad.Read()))

			NewEffect(func() {
				log = append(log, fmt.Sprintf("running nested %d %d", double.Read(), quad.Read()))
				OnCleanup(func() {
					log = append(log, fmt.Sprintf("cleanup nested %d %d", double.Read(), quad.Read()))
				})
			})

			OnCleanup(func() {
				log = append(log, fmt.Sprintf("cleanup %d %d", double.Read(), quad.Read()))
			})
		})

		count.Write(10)

		assert.Equal(t, []string{
			"running 0 0",
			"running nested 0 0",
			"cleanup nested 20 40",
			"cleanup 20 40",
			"running 20 40",
			"running nested 20 40",
		}, log)
	})

	t.Run("deps change between runs", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		initialized := false
		NewEffect(func() {
			log = append(log, "running")
			if !initialized {
				count.Read()
			}
			initialized = true
		})

		count.Write(1)
		count.Write(2) // should not trigger since effect no longer depends on count

		assert.Equal(t, []string{
			"running",
			"running",
		}, log)
	})

	t.Run("goroutine isolation", func(t *testing.T) {
		// run lots of times to make sure it's not a fluke
		for range 1000 {
			var wg sync.WaitGroup

			a := NewSignal(0)
			b := NewSignal(0)

			wg.Go(func() {
				for b.Read() < 5 {
					b.Write(b.Read() + 1)
				}
			})

			calls := 0
			NewEffect(func() {
				a.Read()
				calls++
			})

			a.Write(1)

			wg.Wait()

			if !assert.Equal(t, 2, calls, "effect should only run twice") {
				break
			}
		}
	})

	t.Run("read/write from goroutine", func(t *testing.T) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		log := []int{}
		cleanups := 0

		count := NewSignal(0)

		NewEffect(func() {
			mu.Lock()
			log = append(log, count.Read())
			mu.Unlock()

			OnCleanup(func() {
				mu.Lock()
				cleanups++
				mu.Unlock()
			})
		})

		wg.Go(func() {
			for count.Read() < 5 {
				count.Write(count.Read() + 1)
			}
		})

		wg.Wait()

		assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, log)
		assert.Equal(t, 5, cleanups)
	})
}
