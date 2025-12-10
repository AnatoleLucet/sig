package internal

type Owner struct {
	// cleanup functions to be called when the node is disposed
	cleanups []func()

	// the context values of this owner
	context map[any]any
}

func (r *Runtime) NewOwner() *Owner {
	return &Owner{
		cleanups: make([]func(), 0),
		context:  make(map[any]any),
	}
}
