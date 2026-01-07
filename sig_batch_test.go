package sig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatch(t *testing.T) {
	t.Run("batches multiple writes", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		NewBatch(func() {
			count.Write(10)
			count.Write(20)
			log = append(log, "updated")
		})

		assert.Equal(t, []string{
			"changed 0",
			"updated",
			"cleanup",
			"changed 20",
		}, log)
	})

	t.Run("batches multiple signals", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)
		double := NewSignal(0)

		NewEffect(func() {
			log = append(log, fmt.Sprintf("count %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "count cleanup")
			})
		})

		NewEffect(func() {
			log = append(log, fmt.Sprintf("double %d", double.Read()))

			OnCleanup(func() {
				log = append(log, "double cleanup")
			})
		})

		NewBatch(func() {
			count.Write(10)
			double.Write(count.Read() * 2)
			log = append(log, "updated")
		})

		assert.Equal(t, []string{
			"count 0",
			"double 0",
			"updated",
			"count cleanup",
			"count 10",
			"double cleanup",
			"double 20",
		}, log)
	})

	t.Run("nested batches", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		NewEffect(func() {
			log = append(log, fmt.Sprintf("changed %d", count.Read()))

			OnCleanup(func() {
				log = append(log, "cleanup")
			})
		})

		NewBatch(func() {
			count.Write(10)
			NewBatch(func() {
				count.Write(20)
			})
			log = append(log, "updated")
		})

		assert.Equal(t, []string{
			"changed 0",
			"updated",
			"cleanup",
			"changed 20",
		}, log)
	})
}
