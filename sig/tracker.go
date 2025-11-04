package sig

import (
	"slices"
)

type reactionTracker struct {
	reactions []Reaction
}

func (s *reactionTracker) track(o Observable, r Reaction) {
	if !slices.Contains(s.reactions, r) {
		s.reactions = append(s.reactions, r)
		r.addDependency(o)
	}
}

func (s *reactionTracker) untrack(o Observable, r Reaction) {
	if index := slices.Index(s.reactions, r); index != -1 {
		s.reactions = slices.Delete(s.reactions, index, index+1)
		r.removeDependency(o)
	}
}

func (s *reactionTracker) clear(o Observable) {
	// clonning to avoid mutation during iteration
	reactions := slices.Clone(s.reactions)
	s.reactions = nil

	for _, r := range reactions {
		r.removeDependency(o)
	}
}

func (s *reactionTracker) react(ctx *reactiveContext) {
	// clonning to avoid mutation during iteration
	reactions := slices.Clone(s.reactions)

	for _, r := range reactions {
		ctx.queueReaction(r)
	}
}

type dependencyTracker struct {
	dependencies []Observable
}

func (d *dependencyTracker) add(o Observable) {
	if !slices.Contains(d.dependencies, o) {
		d.dependencies = append(d.dependencies, o)
	}
}

func (d *dependencyTracker) remove(o Observable) {
	if index := slices.Index(d.dependencies, o); index != -1 {
		d.dependencies = slices.Delete(d.dependencies, index, index+1)
	}
}

func (d *dependencyTracker) clear(r Reaction) {
	// clonning to avoid mutation during iteration
	dependencies := slices.Clone(d.dependencies)
	d.dependencies = nil

	for _, dep := range dependencies {
		dep.untrack(r)
	}
}
