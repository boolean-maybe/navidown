package navidown

// NavigationHistory implements a generic browser-like back/forward history stack.
// It maintains two stacks: back (for going backward) and forward (for going forward).
//
// Browser-like behavior:
// - Push adds current state to back stack and clears forward stack
// - Back returns the most recent back entry (caller is responsible for saving current to forward)
// - Forward returns the most recent forward entry (caller is responsible for saving current to back)
// - History is limited to maxSize entries to prevent unbounded memory growth
type NavigationHistory[T any] struct {
	backStack    []T
	forwardStack []T
	maxSize      int
}

func (h *NavigationHistory[T]) trim(slice []T) []T {
	if h.maxSize <= 0 {
		return slice
	}
	if len(slice) <= h.maxSize {
		return slice
	}
	// keep the most recent entries
	return slice[len(slice)-h.maxSize:]
}

// NewNavigationHistory creates a new navigation history with the given maximum size.
func NewNavigationHistory[T any](maxSize int) *NavigationHistory[T] {
	return &NavigationHistory[T]{
		backStack:    make([]T, 0),
		forwardStack: make([]T, 0),
		maxSize:      maxSize,
	}
}

// Push adds a state to the back stack and clears the forward stack.
func (h *NavigationHistory[T]) Push(state T) {
	h.backStack = append(h.backStack, state)
	h.backStack = h.trim(h.backStack)

	h.forwardStack = nil
}

// Back pops from the back stack.
func (h *NavigationHistory[T]) Back() (T, bool) {
	if !h.CanGoBack() {
		var zero T
		return zero, false
	}

	lastIndex := len(h.backStack) - 1
	state := h.backStack[lastIndex]
	h.backStack = h.backStack[:lastIndex]

	return state, true
}

// Forward pops from the forward stack.
func (h *NavigationHistory[T]) Forward() (T, bool) {
	if !h.CanGoForward() {
		var zero T
		return zero, false
	}

	lastIndex := len(h.forwardStack) - 1
	state := h.forwardStack[lastIndex]
	h.forwardStack = h.forwardStack[:lastIndex]

	return state, true
}

// PushToForward adds a state to the forward stack.
func (h *NavigationHistory[T]) PushToForward(state T) {
	h.forwardStack = append(h.forwardStack, state)
	h.forwardStack = h.trim(h.forwardStack)
}

// PushToBack adds a state to the back stack without clearing forward history.
func (h *NavigationHistory[T]) PushToBack(state T) {
	h.backStack = append(h.backStack, state)
	h.backStack = h.trim(h.backStack)
}

// CanGoBack returns true if there are entries in the back stack.
func (h *NavigationHistory[T]) CanGoBack() bool {
	return len(h.backStack) > 0
}

// CanGoForward returns true if there are entries in the forward stack.
func (h *NavigationHistory[T]) CanGoForward() bool {
	return len(h.forwardStack) > 0
}

// Clear removes all history entries.
func (h *NavigationHistory[T]) Clear() {
	h.backStack = nil
	h.forwardStack = nil
}

// BackStackSize returns the number of entries in the back stack.
func (h *NavigationHistory[T]) BackStackSize() int {
	return len(h.backStack)
}

// ForwardStackSize returns the number of entries in the forward stack.
func (h *NavigationHistory[T]) ForwardStackSize() int {
	return len(h.forwardStack)
}
