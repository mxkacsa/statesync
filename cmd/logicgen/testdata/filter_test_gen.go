package filtertest

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/mxkacsa/statesync"
)

var _ = fmt.Sprintf
var _ = math.Sqrt
var _ = rand.Intn
var _ = time.Now

// Permission errors
var (
	ErrNotHost    = errors.New("only host can perform this action")
	ErrNotAllowed = errors.New("player not allowed to perform this action")
)

// ============================================================================
// Filter Factories
// ============================================================================

// HideEnemyLocations creates a filter: Hides enemy team locations from viewer
func HideEnemyLocations(viewerTeam string) statesync.FilterFunc[*GameState] {
	return func(state *GameState) *GameState {
		// Clone state for safe modification
		filtered := state.ShallowClone()

		// TODO: Filter node logic would be generated here
		// Nodes: 0, Flow edges: 0

		return filtered
	}
}

// HidePlayerHand creates a filter: Hides a player's private data from others
func HidePlayerHand(playerID string, viewerID string) statesync.FilterFunc[*GameState] {
	return func(state *GameState) *GameState {
		// Clone state for safe modification
		filtered := state.ShallowClone()

		// TODO: Filter node logic would be generated here
		// Nodes: 0, Flow edges: 0

		return filtered
	}
}

// ============================================================================
// Filter Registry
// ============================================================================

// FilterRegistry manages active filters per viewer
type FilterRegistry struct {
	mu      sync.RWMutex
	filters map[string]map[string]statesync.FilterFunc[*GameState]
}

// NewFilterRegistry creates a new filter registry
func NewFilterRegistry() *FilterRegistry {
	return &FilterRegistry{
		filters: make(map[string]map[string]statesync.FilterFunc[*GameState]),
	}
}

// Add adds a filter instance for a viewer
func (r *FilterRegistry) Add(viewerID, filterID string, filter statesync.FilterFunc[*GameState]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.filters[viewerID] == nil {
		r.filters[viewerID] = make(map[string]statesync.FilterFunc[*GameState])
	}
	r.filters[viewerID][filterID] = filter
}

// Remove removes a filter instance
func (r *FilterRegistry) Remove(viewerID, filterID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.filters[viewerID] == nil {
		return false
	}
	_, ok := r.filters[viewerID][filterID]
	delete(r.filters[viewerID], filterID)
	return ok
}

// Has checks if a filter exists
func (r *FilterRegistry) Has(viewerID, filterID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.filters[viewerID] == nil {
		return false
	}
	_, ok := r.filters[viewerID][filterID]
	return ok
}

// GetComposed returns a composed filter for a viewer
func (r *FilterRegistry) GetComposed(viewerID string) statesync.FilterFunc[*GameState] {
	r.mu.RLock()
	filters := r.filters[viewerID]
	if len(filters) == 0 {
		r.mu.RUnlock()
		return nil
	}
	fns := make([]statesync.FilterFunc[*GameState], 0, len(filters))
	for _, f := range filters {
		fns = append(fns, f)
	}
	r.mu.RUnlock()

	return func(state *GameState) *GameState {
		for _, fn := range fns {
			state = fn(state)
		}
		return state
	}
}

// Global filter registry - initialize with NewFilterRegistry()
var filterRegistry *FilterRegistry

// OnPlayerJoin handles the PlayerJoin event
func OnPlayerJoin(session *statesync.TrackedSession[*GameState, any, string], senderID string, playerID string, team string) error {
	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: addEnemyFilter (AddFilter)
	addEnemyFilter_filter := HideEnemyLocations(team)
	filterRegistry.Add(playerID, playerID, addEnemyFilter_filter)
	session.SetFilter(playerID, filterRegistry.GetComposed(playerID))
	// Node: emitJoined (EmitToAll)
	session.Emit("PlayerJoined", nil)
	return nil
}

// OnPlayerLeave handles the PlayerLeave event
func OnPlayerLeave(session *statesync.TrackedSession[*GameState, any, string], senderID string, playerID string) error {
	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: removeEnemyFilter (RemoveFilter)
	removeEnemyFilter_removed := filterRegistry.Remove(playerID, playerID)
	session.SetFilter(playerID, filterRegistry.GetComposed(playerID))
	return nil
}

// OnCheckFilter handles the CheckFilter event
func OnCheckFilter(session *statesync.TrackedSession[*GameState, any, string], senderID string, playerID string, filterID string) error {
	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: hasFilter (HasFilter)
	hasFilter_exists := filterRegistry.Has(playerID, filterID)
	// Node: emitResult (EmitToAll)
	session.Emit("FilterCheckResult", nil)
	return nil
}
