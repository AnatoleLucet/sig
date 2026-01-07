package sig

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignal(t *testing.T) {
	t.Run("read and write", func(t *testing.T) {
		count := NewSignal(0)
		assert.Equal(t, 0, count.Read())

		count.Write(10)
		assert.Equal(t, 10, count.Read())
	})

	t.Run("concurrent read/write", func(t *testing.T) {
		var wg sync.WaitGroup
		count := NewSignal(0)

		wg.Go(func() {
			count.Write(count.Read() + 1)
		})

		wg.Wait()
		assert.Equal(t, 1, count.Read())
	})

	t.Run("zero values", func(t *testing.T) {
		err := NewSignal[error](nil)
		assert.Nil(t, err.Read())

		err.Write(errors.New("oops"))
		assert.EqualError(t, err.Read(), "oops")

		err.Write(nil)
		assert.Nil(t, err.Read())
	})
}
