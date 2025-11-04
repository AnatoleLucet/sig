package proto

// flags represents the state of a reactive node
type flags uint8

const (
	FlagNone        flags = 0
	FlagCheck       flags = 1 << iota // Node might need recomputation (check deps first)
	FlagDirty                         // Node definitely needs recomputation
	FlagInHeap                        // Node is currently in the dirty heap
	FlagRecomputing                   // Node is currently being recomputed
)

func (f flags) has(flag flags) bool {
	return f&flag != 0
}

func (f *flags) set(flag flags) {
	*f |= flag
}

func (f *flags) clear(flag flags) {
	*f &^= flag
}

func (f *flags) replace(old, new flags) {
	*f = (*f &^ old) | new
}
