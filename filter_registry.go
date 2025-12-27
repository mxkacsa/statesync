package statesync

import (
	"sync"
)

// FilterRegistry manages active filters per viewer.
// Filters can be dynamically added/removed at runtime.
type FilterRegistry[T any, ID comparable] struct {
	mu      sync.RWMutex
	filters map[ID]map[string]FilterFunc[T] // viewerID -> filterID -> filter
}

// NewFilterRegistry creates a new filter registry
func NewFilterRegistry[T any, ID comparable]() *FilterRegistry[T, ID] {
	return &FilterRegistry[T, ID]{
		filters: make(map[ID]map[string]FilterFunc[T]),
	}
}

// Add adds a filter for a viewer
func (r *FilterRegistry[T, ID]) Add(viewerID ID, filterID string, filter FilterFunc[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.filters[viewerID] == nil {
		r.filters[viewerID] = make(map[string]FilterFunc[T])
	}
	r.filters[viewerID][filterID] = filter
}

// Remove removes a filter from a viewer
func (r *FilterRegistry[T, ID]) Remove(viewerID ID, filterID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.filters[viewerID] == nil {
		return false
	}

	_, ok := r.filters[viewerID][filterID]
	if ok {
		delete(r.filters[viewerID], filterID)
		// Clean up empty maps
		if len(r.filters[viewerID]) == 0 {
			delete(r.filters, viewerID)
		}
	}
	return ok
}

// Has checks if a filter exists for a viewer
func (r *FilterRegistry[T, ID]) Has(viewerID ID, filterID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.filters[viewerID] == nil {
		return false
	}
	_, ok := r.filters[viewerID][filterID]
	return ok
}

// Get returns a specific filter (nil if not found)
func (r *FilterRegistry[T, ID]) Get(viewerID ID, filterID string) FilterFunc[T] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.filters[viewerID] == nil {
		return nil
	}
	return r.filters[viewerID][filterID]
}

// GetAll returns all filter IDs for a viewer
func (r *FilterRegistry[T, ID]) GetAll(viewerID ID) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.filters[viewerID] == nil {
		return nil
	}

	ids := make([]string, 0, len(r.filters[viewerID]))
	for id := range r.filters[viewerID] {
		ids = append(ids, id)
	}
	return ids
}

// Clear removes all filters for a viewer
func (r *FilterRegistry[T, ID]) Clear(viewerID ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.filters, viewerID)
}

// ClearAll removes all filters for all viewers
func (r *FilterRegistry[T, ID]) ClearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filters = make(map[ID]map[string]FilterFunc[T])
}

// Count returns the number of filters for a viewer
func (r *FilterRegistry[T, ID]) Count(viewerID ID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.filters[viewerID] == nil {
		return 0
	}
	return len(r.filters[viewerID])
}

// Compose returns a single filter that applies all filters for a viewer
func (r *FilterRegistry[T, ID]) Compose(viewerID ID) FilterFunc[T] {
	r.mu.RLock()
	viewerFilters := r.filters[viewerID]
	if len(viewerFilters) == 0 {
		r.mu.RUnlock()
		return nil
	}

	// Copy filters to avoid holding lock during execution
	fns := make([]FilterFunc[T], 0, len(viewerFilters))
	for _, f := range viewerFilters {
		fns = append(fns, f)
	}
	r.mu.RUnlock()

	// Single filter - no composition needed
	if len(fns) == 1 {
		return fns[0]
	}

	// Multiple filters - compose them
	return func(state T) T {
		result := state
		for _, fn := range fns {
			result = fn(result)
		}
		return result
	}
}

// ComposeWith returns a filter that first applies the base filter, then all registered filters
func (r *FilterRegistry[T, ID]) ComposeWith(viewerID ID, base FilterFunc[T]) FilterFunc[T] {
	composed := r.Compose(viewerID)

	if base == nil && composed == nil {
		return nil
	}
	if base == nil {
		return composed
	}
	if composed == nil {
		return base
	}

	return func(state T) T {
		return composed(base(state))
	}
}
