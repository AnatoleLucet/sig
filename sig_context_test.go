package sig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	t.Run("store value", func(t *testing.T) {
		ctx := NewContext(0)
		assert.Equal(t, 0, ctx.Value())

		ctx.Set(42)
		assert.Equal(t, 0, ctx.Value()) // still zero, no owner to hold the value
	})

	t.Run("inherit value from parent owner", func(t *testing.T) {
		ctx := NewContext("default")

		parent := NewOwner()
		err := parent.Run(func() error {
			ctx.Set("parent value")

			return NewOwner().Run(func() error {
				assert.Equal(t, "parent value", ctx.Value())
				return nil
			})
		})
		assert.NoError(t, err)

		assert.Equal(t, "default", ctx.Value())
	})
}
