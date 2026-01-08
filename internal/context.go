package internal

type contextID struct{}

type Context struct {
	id           *contextID
	defaultValue any
}

func (r *Runtime) NewContext(defaultValue any) *Context {
	return &Context{
		id:           &contextID{},
		defaultValue: defaultValue,
	}
}

func (c *Context) Value() any {
	owner := GetRuntime().CurrentOwner()

	for o := owner; o != nil; o = o.parent {
		if val, ok := o.context[c.id]; ok {
			return val
		}
	}

	return c.defaultValue
}

func (c *Context) Set(value any) {
	owner := GetRuntime().CurrentOwner()

	if owner != nil {
		owner.context[c.id] = value
	}
}
