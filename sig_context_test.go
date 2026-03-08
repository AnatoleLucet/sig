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

		err := NewOwner().Run(func() error {
			ctx.Set("parent value")

			return NewOwner().Run(func() error {
				assert.Equal(t, "parent value", ctx.Value())
				return nil
			})
		})
		assert.NoError(t, err)

		assert.Equal(t, "default", ctx.Value())
	})

	t.Run("interleaving contexts", func(t *testing.T) {
		ctx1 := NewContext("ctx1 default")
		ctx2 := NewContext("ctx2 default")

		err := NewOwner().Run(func() error {
			ctx1.Set("ctx1 value")

			assert.Equal(t, "ctx1 value", ctx1.Value())
			assert.Equal(t, "ctx2 default", ctx2.Value())

			return NewOwner().Run(func() error {
				ctx2.Set("ctx2 value")

				assert.Equal(t, "ctx1 value", ctx1.Value())
				assert.Equal(t, "ctx2 value", ctx2.Value())
				return nil
			})
		})

		assert.NoError(t, err)
		assert.Equal(t, "ctx1 default", ctx1.Value())
		assert.Equal(t, "ctx2 default", ctx2.Value())
	})
}
