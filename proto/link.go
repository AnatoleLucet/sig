package proto

// link represents a connection between a dependency (observable) and a subscriber (reaction)
type link struct {
	dep Observable // The dependency being tracked
	sub Reaction   // The reaction subscribing to the dependency

	// Doubly-linked list pointers for the subscriber's dependency list
	nextDep *link
	prevDep *link

	// Doubly-linked list pointers for the dependency's subscription list
	nextSub *link
	prevSub *link
}

// linkDep connects a dependency to a subscriber
func linkDep(dep Observable, sub Reaction) {
	subNode := sub.node()
	depNode := dep.node()

	// Check if already linked (avoid duplicates)
	if subNode.depsTail != nil && subNode.depsTail.dep == dep {
		return
	}

	// During recomputation, check if we're reusing an existing link
	var nextDep *link
	if subNode.flags.has(FlagRecomputing) {
		if subNode.depsTail != nil {
			nextDep = subNode.depsTail.nextDep
		} else {
			nextDep = subNode.deps
		}

		if nextDep != nil && nextDep.dep == dep {
			subNode.depsTail = nextDep
			return
		}
	}

	// Create new link
	newLink := &link{
		dep:     dep,
		sub:     sub,
		nextDep: nextDep,
		prevSub: depNode.subsTail,
	}

	// Add to subscriber's dependency list
	if subNode.depsTail != nil {
		subNode.depsTail.nextDep = newLink
		newLink.prevDep = subNode.depsTail
	} else {
		subNode.deps = newLink
	}
	subNode.depsTail = newLink

	// Add to dependency's subscription list
	if depNode.subsTail != nil {
		depNode.subsTail.nextSub = newLink
	} else {
		depNode.subs = newLink
	}
	depNode.subsTail = newLink
}

// unlinkSub removes a link and returns the next dependency link
func unlinkSub(l *link) *link {
	dep := l.dep.node()
	nextDep := l.nextDep

	// Remove from subscription list
	if l.nextSub != nil {
		l.nextSub.prevSub = l.prevSub
	} else {
		dep.subsTail = l.prevSub
	}

	if l.prevSub != nil {
		l.prevSub.nextSub = l.nextSub
	} else {
		dep.subs = l.nextSub
	}

	return nextDep
}
