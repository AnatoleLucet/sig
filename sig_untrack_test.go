package sig

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUntrack(t *testing.T) {
	t.Run("does not track reads", func(t *testing.T) {
		log := []string{}

		count := NewSignal(0)

		NewEffect(func() {
			c := Untrack(count.Read)
			log = append(log, fmt.Sprintf("effect %d", c))
		})

		count.Write(10)

		assert.Equal(t, []string{
			"effect 0",
		}, log)
	})
}
