package sigv2

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// Context represents a typed context that can be passed through the reactive tree
type Context[T any] struct {
	id           interface{}
	defaultValue *T
	description  string
}

// CreateContext creates a new context with an optional default value
func CreateContext[T any](defaultValue *T, description string) *Context[T] {
	// Generate unique ID
	var id [16]byte
	rand.Read(id[:])

	return &Context[T]{
		id:           id,
		defaultValue: defaultValue,
		description:  description,
	}
}

// GetContext retrieves the context value from the current owner
func GetContext[T any](ctx *Context[T], owner ...*ownerNode) (T, error) {
	var o *ownerNode
	if len(owner) > 0 && owner[0] != nil {
		o = owner[0]
	} else {
		o = getOwner()
	}

	if o == nil {
		var zero T
		return zero, &NoOwnerError{}
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	// Check if context exists
	if value, ok := o.contextMap[ctx.id]; ok {
		if v, ok := value.(T); ok {
			return v, nil
		}
	}

	// Return default value if available
	if ctx.defaultValue != nil {
		return *ctx.defaultValue, nil
	}

	var zero T
	return zero, &ContextNotFoundError{}
}

// SetContext sets a context value in the current owner
func SetContext[T any](ctx *Context[T], value T, owner ...*ownerNode) error {
	var o *ownerNode
	if len(owner) > 0 && owner[0] != nil {
		o = owner[0]
	} else {
		o = getOwner()
	}

	if o == nil {
		return &NoOwnerError{}
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	// Initialize context map if needed
	if o.contextMap == nil {
		o.contextMap = make(map[interface{}]interface{})
	}

	o.contextMap[ctx.id] = value
	return nil
}

// HasContext checks if a context value exists in the owner
func HasContext[T any](ctx *Context[T], owner *ownerNode) bool {
	if owner == nil {
		return false
	}

	owner.mu.RLock()
	defer owner.mu.RUnlock()

	_, ok := owner.contextMap[ctx.id]
	return ok
}

// generateSymbolID creates a unique identifier similar to JavaScript Symbol
func generateSymbolID() interface{} {
	var b [8]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint64(b[:])
}

// String returns a string representation of the context
func (c *Context[T]) String() string {
	if c.description != "" {
		return fmt.Sprintf("Context(%s)", c.description)
	}
	return fmt.Sprintf("Context(%v)", c.id)
}
