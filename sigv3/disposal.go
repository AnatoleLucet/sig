package sig

type Disposal struct {
	effect func()
	user   []func()
}

func NewDisposal() *Disposal {
	return  &Disposal{
		user: make([]func(), 0)
	}
}

func (d *Disposal) SetEffectCleanup(cleanup func()) {
	d.effect = cleanup
}

func (d *Disposal) AddUserCleanup(cleanup func()) {
	d.user = append(d.user, cleanup)
}

func (d *Disposal) Clear() {
	d.effect = nil
	d.user = nil
}

// Run executes the effect cleanup, then the user defined cleanups
func (d *Disposal) Run() {
	d.RunEffect()
	d.RunUser()
}

// RunEffect executes the effect cleanup
func (d *Disposal) RunEffect() {
	if d.effect != nil {
		d.effect()
		d.effect = nil
	}
}

// RunUser executes the user cleanups
func (d *Disposal) RunUser() {
	for _, cleanup := range d.user {
		cleanup()
	}
	d.user = nil
}
