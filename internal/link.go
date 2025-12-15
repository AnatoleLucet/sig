package internal

type DependencyLink struct {
	dep *Signal
	sub *Computed

	prevDep *DependencyLink
	nextDep *DependencyLink

	prevSub *DependencyLink
	nextSub *DependencyLink
}
