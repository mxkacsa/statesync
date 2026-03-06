package statesync

import (
	"math/bits"
	"sync"
)

// Operation types for change tracking
type Operation uint8

const (
	OpNone    Operation = iota
	OpAdd               // New value added (array/map)
	OpReplace           // Value replaced
	OpRemove            // Value removed (array/map)
	OpMove              // Element moved (array reorder)
)

func (o Operation) String() string {
	switch o {
	case OpNone:
		return "none"
	case OpAdd:
		return "add"
	case OpReplace:
		return "replace"
	case OpRemove:
		return "remove"
	case OpMove:
		return "move"
	default:
		return "unknown"
	}
}

// FieldChange represents a change to a single field
type FieldChange struct {
	Op       Operation
	OldIndex int // For array operations (move/remove source)
	NewIndex int // For array operations (move/add target)
}

// ChangeSet tracks changes to an object's fields
type ChangeSet struct {
	mu sync.RWMutex

	// Bitset for tracking which fields changed (supports up to 256 fields)
	// Each bit represents a field index
	dirty [4]uint64

	// Field operations (indexed by field number, only valid if dirty bit is set)
	ops [256]FieldChange

	// For nested objects: field index -> child ChangeSet
	children map[uint8]*ChangeSet

	// For arrays: field index -> array changes
	arrays map[uint8]*ArrayChangeSet

	// For maps: field index -> map changes
	maps map[uint8]*MapChangeSet
}

// ArrayChangeSet tracks changes to array elements
type ArrayChangeSet struct {
	// Index -> operation for changed elements
	changes map[int]ArrayElementChange
	// Track length changes
	oldLen, newLen int
}

// ArrayElementChange represents a change to an array element
type ArrayElementChange struct {
	Op       Operation
	OldIndex int         // Original index (for moves)
	Value    interface{} // New value (for add/replace)
}

// MapChangeSet tracks changes to map entries
type MapChangeSet struct {
	// Key -> operation for changed entries
	changes map[string]MapEntryChange
}

// MapEntryChange represents a change to a map entry
type MapEntryChange struct {
	Op    Operation
	Value interface{} // New value (for add/replace)
}

// NewChangeSet creates an empty ChangeSet
func NewChangeSet() *ChangeSet {
	return &ChangeSet{}
}

// Mark marks a field as changed
func (cs *ChangeSet) Mark(fieldIndex uint8, op Operation) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	// Set bit in dirty bitset
	cs.dirty[fieldIndex/64] |= 1 << (fieldIndex % 64)
	cs.ops[fieldIndex] = FieldChange{Op: op}
}

// MarkWithIndex marks an array field change with index info
func (cs *ChangeSet) MarkWithIndex(fieldIndex uint8, op Operation, oldIdx, newIdx int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.dirty[fieldIndex/64] |= 1 << (fieldIndex % 64)
	cs.ops[fieldIndex] = FieldChange{Op: op, OldIndex: oldIdx, NewIndex: newIdx}
}

// GetFieldChange returns the change for a field, or OpNone if unchanged
func (cs *ChangeSet) GetFieldChange(fieldIndex uint8) FieldChange {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if cs.isDirty(fieldIndex) {
		return cs.ops[fieldIndex]
	}
	return FieldChange{Op: OpNone}
}

// IsFieldDirty returns true if the field has been changed (fast bitset check)
func (cs *ChangeSet) IsFieldDirty(fieldIndex uint8) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.isDirty(fieldIndex)
}

// HasChanges returns true if any fields have changed
func (cs *ChangeSet) HasChanges() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if cs.dirty[0] != 0 || cs.dirty[1] != 0 || cs.dirty[2] != 0 || cs.dirty[3] != 0 {
		return true
	}
	for _, child := range cs.children {
		if child.HasChanges() {
			return true
		}
	}
	for _, arr := range cs.arrays {
		if arr.HasChanges() {
			return true
		}
	}
	for _, m := range cs.maps {
		if m.HasChanges() {
			return true
		}
	}
	return false
}

// Clear removes all tracked changes
func (cs *ChangeSet) Clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	// Clear bitset (zero allocation)
	cs.dirty[0] = 0
	cs.dirty[1] = 0
	cs.dirty[2] = 0
	cs.dirty[3] = 0
	// Recursively clear nested objects (don't delete from map — cached pointers stay valid)
	for _, child := range cs.children {
		child.Clear()
	}
	// Clear array and map change sets
	clear(cs.arrays)
	clear(cs.maps)
}

// CloneForFilter creates a copy of the ChangeSet suitable for use in projection filters.
// The clone has its own maps/arrays tracking so that filter modifications (e.g., DeleteCollectiblesKey)
// don't corrupt the original ChangeSet that other filters will read.
// Scalar dirty bits and ops are copied by value (cheap). Map/array change sets are deep-copied.
func (cs *ChangeSet) CloneForFilter() *ChangeSet {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	clone := &ChangeSet{
		dirty: cs.dirty,
		ops:   cs.ops,
	}

	// Deep-copy map change sets
	if len(cs.maps) > 0 {
		clone.maps = make(map[uint8]*MapChangeSet, len(cs.maps))
		for idx, mcs := range cs.maps {
			if mcs == nil {
				continue
			}
			clonedMcs := &MapChangeSet{changes: make(map[string]MapEntryChange, len(mcs.changes))}
			for k, v := range mcs.changes {
				clonedMcs.changes[k] = v
			}
			clone.maps[idx] = clonedMcs
		}
	}

	// Deep-copy array change sets
	if len(cs.arrays) > 0 {
		clone.arrays = make(map[uint8]*ArrayChangeSet, len(cs.arrays))
		for idx, acs := range cs.arrays {
			if acs == nil {
				continue
			}
			clonedAcs := &ArrayChangeSet{
				changes: make(map[int]ArrayElementChange, len(acs.changes)),
				oldLen:  acs.oldLen,
				newLen:  acs.newLen,
			}
			for k, v := range acs.changes {
				clonedAcs.changes[k] = v
			}
			clone.arrays[idx] = clonedAcs
		}
	}

	// Note: children (nested struct ChangeSets) are NOT deep-copied.
	// Projections don't modify nested struct schemas directly.

	return clone
}

// MarkAll marks all fields up to maxIndex as changed (for full sync)
func (cs *ChangeSet) MarkAll(maxIndex uint8) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for i := 0; i <= int(maxIndex); i++ {
		idx := uint8(i)
		cs.dirty[idx/64] |= 1 << (idx % 64)
		cs.ops[idx] = FieldChange{Op: OpReplace}
	}
}

// GetOrCreateChild returns the ChangeSet for a nested object, creating if needed
func (cs *ChangeSet) GetOrCreateChild(fieldIndex uint8) *ChangeSet {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.children == nil {
		cs.children = make(map[uint8]*ChangeSet)
	}
	if child, ok := cs.children[fieldIndex]; ok {
		return child
	}
	child := NewChangeSet()
	cs.children[fieldIndex] = child
	return child
}

// GetChild returns the ChangeSet for a nested object, or nil if none
func (cs *ChangeSet) GetChild(fieldIndex uint8) *ChangeSet {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.children[fieldIndex]
}

// GetOrCreateArray returns the ArrayChangeSet for an array field
func (cs *ChangeSet) GetOrCreateArray(fieldIndex uint8) *ArrayChangeSet {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.arrays == nil {
		cs.arrays = make(map[uint8]*ArrayChangeSet)
	}
	if arr, ok := cs.arrays[fieldIndex]; ok {
		return arr
	}
	arr := &ArrayChangeSet{changes: make(map[int]ArrayElementChange)}
	cs.arrays[fieldIndex] = arr
	return arr
}

// GetArray returns the ArrayChangeSet for an array field, or nil
func (cs *ChangeSet) GetArray(fieldIndex uint8) *ArrayChangeSet {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.arrays[fieldIndex]
}

// GetOrCreateMap returns the MapChangeSet for a map field
func (cs *ChangeSet) GetOrCreateMap(fieldIndex uint8) *MapChangeSet {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.maps == nil {
		cs.maps = make(map[uint8]*MapChangeSet)
	}
	if m, ok := cs.maps[fieldIndex]; ok {
		return m
	}
	m := &MapChangeSet{changes: make(map[string]MapEntryChange)}
	cs.maps[fieldIndex] = m
	return m
}

// GetMap returns the MapChangeSet for a map field, or nil
func (cs *ChangeSet) GetMap(fieldIndex uint8) *MapChangeSet {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.maps[fieldIndex]
}

// ChangedFields returns all changed field indices in sorted order
func (cs *ChangeSet) ChangedFields() []uint8 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.changedFieldsLocked()
}

// changedFieldsLocked returns changed fields without locking (caller must hold lock)
func (cs *ChangeSet) changedFieldsLocked() []uint8 {
	// Count dirty bits for pre-allocation
	count := popcount(cs.dirty[0]) + popcount(cs.dirty[1]) + popcount(cs.dirty[2]) + popcount(cs.dirty[3])
	if count == 0 && len(cs.children) == 0 && len(cs.arrays) == 0 && len(cs.maps) == 0 {
		return nil
	}

	result := make([]uint8, 0, count)

	// Extract set bits from bitset (already in sorted order)
	for i := 0; i < 4; i++ {
		w := cs.dirty[i]
		base := uint8(i * 64)
		for w != 0 {
			tz := trailingZeros64(w)
			result = append(result, base+uint8(tz))
			w &= w - 1
		}
	}

	// Add any children/arrays/maps whose parent dirty bit is not set
	for idx := range cs.children {
		if !cs.isDirty(idx) {
			result = append(result, idx)
		}
	}
	for idx, arr := range cs.arrays {
		if !cs.isDirty(idx) && arr.HasChanges() {
			result = append(result, idx)
		}
	}
	for idx, m := range cs.maps {
		if !cs.isDirty(idx) && m.HasChanges() {
			result = append(result, idx)
		}
	}

	return result
}

// isDirty checks if a field's dirty bit is set (caller must hold lock)
func (cs *ChangeSet) isDirty(idx uint8) bool {
	return cs.dirty[idx/64]&(1<<(idx%64)) != 0
}

// popcount returns the number of set bits in x (uses hardware POPCNT instruction)
func popcount(x uint64) int {
	return bits.OnesCount64(x)
}

// trailingZeros64 returns the number of trailing zero bits (uses hardware TZCNT instruction)
func trailingZeros64(x uint64) int {
	return bits.TrailingZeros64(x)
}

// ArrayChangeSet methods

// MarkAdd marks an element addition at index
func (acs *ArrayChangeSet) MarkAdd(index int, value interface{}) {
	acs.changes[index] = ArrayElementChange{Op: OpAdd, Value: value}
}

// MarkReplace marks an element replacement at index
func (acs *ArrayChangeSet) MarkReplace(index int, value interface{}) {
	acs.changes[index] = ArrayElementChange{Op: OpReplace, Value: value}
}

// MarkRemove marks an element removal at index
func (acs *ArrayChangeSet) MarkRemove(index int) {
	acs.changes[index] = ArrayElementChange{Op: OpRemove, OldIndex: index}
}

// MarkMove marks an element move from oldIdx to newIdx
func (acs *ArrayChangeSet) MarkMove(oldIdx, newIdx int) {
	acs.changes[newIdx] = ArrayElementChange{Op: OpMove, OldIndex: oldIdx}
}

// HasChanges returns true if array has changes
func (acs *ArrayChangeSet) HasChanges() bool {
	return len(acs.changes) > 0
}

// Clear removes all array changes
func (acs *ArrayChangeSet) Clear() {
	acs.changes = make(map[int]ArrayElementChange)
}

// MapChangeSet methods

// MarkAdd marks a map entry addition
func (mcs *MapChangeSet) MarkAdd(key string, value interface{}) {
	mcs.changes[key] = MapEntryChange{Op: OpAdd, Value: value}
}

// MarkReplace marks a map entry replacement
func (mcs *MapChangeSet) MarkReplace(key string, value interface{}) {
	mcs.changes[key] = MapEntryChange{Op: OpReplace, Value: value}
}

// MarkRemove marks a map entry removal
func (mcs *MapChangeSet) MarkRemove(key string) {
	mcs.changes[key] = MapEntryChange{Op: OpRemove}
}

// HasChanges returns true if map has changes
func (mcs *MapChangeSet) HasChanges() bool {
	return len(mcs.changes) > 0
}

// Clear removes all map changes
func (mcs *MapChangeSet) Clear() {
	mcs.changes = make(map[string]MapEntryChange)
}
