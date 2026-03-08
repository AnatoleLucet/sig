package sig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputed(t *testing.T) {
	t.Run("derives value from signal", func(t *testing.T) {
		log := []string{}

		count := NewSignal(1)
		double := NewComputed(func() int {
			log = append(log, "doubling")
			return count.Read() * 2
		})
		plustwo := NewComputed(func() int {
			log = append(log, "adding")
			return double.Read() + 2
		})

		assert.Equal(t, 1, count.Read())
		assert.Equal(t, 2, double.Read())
		assert.Equal(t, 4, plustwo.Read())

		count.Write(10)
		assert.Equal(t, 10, count.Read())
		assert.Equal(t, 20, double.Read())
		assert.Equal(t, 22, plustwo.Read())

		assert.Equal(t, []string{
			"doubling",
			"adding",
			"doubling",
			"adding",
		}, log)
	})

	t.Run("does not propagate when value unchanged", func(t *testing.T) {
		log := []string{}

		count := NewSignal(1)
		a := NewComputed(func() int {
			log = append(log, "running a")
			return count.Read() * 0 // always returns 0
		})
		b := NewComputed(func() int {
			log = append(log, "running b")
			return a.Read() + 1
		})

		a.Read()
		b.Read()

		count.Write(10) // should recompute a but not b since a's value didn't change

		assert.Equal(t, []string{
			"running a",
			"running b",
			"running a",
		}, log)
	})

	t.Run("custom predicate prevents propagation", func(t *testing.T) {
		log := []string{}

		count := NewSignal(1, SignalOptions[int]{
			Predicate: func(a, b int) bool {
				log = append(log, fmt.Sprintf("comparing %d and %d", a, b))
				return b%2 == 0 // propagate only if new value is odd
			},
		})
		double := NewComputed(func() int {
			log = append(log, "computing double")
			return count.Read() * 2
		})

		assert.Equal(t, 1, count.Read())
		assert.Equal(t, 2, double.Read())

		count.Write(2) // shouldn't propagate
		count.Write(3)
		count.Write(4) // shouldn't propagate

		assert.Equal(t, []string{
			"computing double",
			"comparing 1 and 2",
			"comparing 1 and 3", // should still be at 1 since 2 didn't propagate
			"computing double",
			"comparing 3 and 4",
		}, log)
	})

	t.Run("disposes nested effects on recompute", func(t *testing.T) {
		t.Skip("WIP")

		log := []string{}

		count := NewSignal(1)
		double := NewComputed(func() int {
			log = append(log, "computing")

			NewEffect(func() {
				log = append(log, fmt.Sprintf("effect %d", count.Read()))

				OnCleanup(func() {
					log = append(log, fmt.Sprintf("cleanup %d", count.Read()))
				})
			})

			return count.Read() * 2
		})

		log = append(log, fmt.Sprintf("%d", double.Read()))

		count.Write(10)
		log = append(log, fmt.Sprintf("%d", double.Read()))

		// TODO: define expected behavior
		_ = log
	})
}
