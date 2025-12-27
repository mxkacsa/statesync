package statesync

import (
	"testing"
)

// Test state with team-based chat
type FilterTestState struct {
	changes *ChangeSet
	Players []FilterTestPlayer
	Chat    []FilterTestMessage
}

type FilterTestPlayer struct {
	ID   string
	Team string
}

type FilterTestMessage struct {
	ID   string
	Team string
	Text string
}

func NewFilterTestState() *FilterTestState {
	return &FilterTestState{
		changes: NewChangeSet(),
		Players: make([]FilterTestPlayer, 0),
		Chat:    make([]FilterTestMessage, 0),
	}
}

func (s *FilterTestState) Changes() *ChangeSet { return s.changes }
func (s *FilterTestState) ClearChanges()       { s.changes.Clear() }
func (s *FilterTestState) MarkAllDirty()       { s.changes.MarkAll(1) }
func (s *FilterTestState) Schema() *Schema {
	return &Schema{
		ID:   50,
		Name: "FilterTestState",
		Fields: []FieldMeta{
			{Index: 0, Name: "Players", Type: TypeArray, ElemType: TypeStruct},
			{Index: 1, Name: "Chat", Type: TypeArray, ElemType: TypeStruct},
		},
	}
}

func (s *FilterTestState) GetFieldValue(index uint8) interface{} {
	switch index {
	case 0:
		return s.Players
	case 1:
		return s.Chat
	default:
		return nil
	}
}

func (s *FilterTestState) ShallowClone() *FilterTestState {
	return &FilterTestState{
		changes: s.changes,
		Players: s.Players,
		Chat:    s.Chat,
	}
}

// TeamChatFilter filters chat messages by team
func createTeamChatFilter(viewerTeam string) FilterFunc[*FilterTestState] {
	return func(state *FilterTestState) *FilterTestState {
		filtered := state.ShallowClone()

		// Filter chat to only show viewer's team messages
		filteredChat := make([]FilterTestMessage, 0)
		for _, msg := range state.Chat {
			if msg.Team == viewerTeam {
				filteredChat = append(filteredChat, msg)
			}
		}

		filtered.Chat = filteredChat
		return filtered
	}
}

func TestTeamChatFilter_Basic(t *testing.T) {
	state := NewFilterTestState()

	// Add players
	state.Players = []FilterTestPlayer{
		{ID: "p1", Team: "catcher"},
		{ID: "p2", Team: "runner"},
	}

	// Add chat messages from both teams
	state.Chat = []FilterTestMessage{
		{ID: "m1", Team: "catcher", Text: "Catcher message 1"},
		{ID: "m2", Team: "runner", Text: "Runner message 1"},
		{ID: "m3", Team: "catcher", Text: "Catcher message 2"},
		{ID: "m4", Team: "runner", Text: "Runner message 2"},
	}

	// Test catcher filter
	catcherFilter := createTeamChatFilter("catcher")
	catcherView := catcherFilter(state)

	if len(catcherView.Chat) != 2 {
		t.Errorf("expected 2 catcher messages, got %d", len(catcherView.Chat))
	}

	for _, msg := range catcherView.Chat {
		if msg.Team != "catcher" {
			t.Errorf("catcher should not see runner message: %s", msg.Text)
		}
	}

	// Test runner filter
	runnerFilter := createTeamChatFilter("runner")
	runnerView := runnerFilter(state)

	if len(runnerView.Chat) != 2 {
		t.Errorf("expected 2 runner messages, got %d", len(runnerView.Chat))
	}

	for _, msg := range runnerView.Chat {
		if msg.Team != "runner" {
			t.Errorf("runner should not see catcher message: %s", msg.Text)
		}
	}
}

func TestTeamChatFilter_WithSession(t *testing.T) {
	state := NewFilterTestState()

	// Add players and messages
	state.Players = []FilterTestPlayer{
		{ID: "catcher1", Team: "catcher"},
		{ID: "runner1", Team: "runner"},
	}
	state.Chat = []FilterTestMessage{
		{ID: "m1", Team: "catcher", Text: "Secret catcher strategy"},
		{ID: "m2", Team: "runner", Text: "Secret runner strategy"},
	}
	state.MarkAllDirty()

	// Create session
	tracked := NewTrackedState[*FilterTestState, any](state, nil)
	session := NewTrackedSession[*FilterTestState, any, string](tracked)

	// Connect players with filters
	session.Connect("catcher1", createTeamChatFilter("catcher"))
	session.Connect("runner1", createTeamChatFilter("runner"))

	// Broadcast - each player gets filtered data
	diffs := session.Tick()

	// Both should receive diffs
	if len(diffs) != 2 {
		t.Errorf("expected 2 diffs, got %d", len(diffs))
	}

	// Verify filter is applied
	catcherFilter := session.GetFilter("catcher1")
	runnerFilter := session.GetFilter("runner1")

	if catcherFilter == nil || runnerFilter == nil {
		t.Fatal("filters should be set")
	}

	catcherState := catcherFilter(state)
	runnerState := runnerFilter(state)

	if len(catcherState.Chat) != 1 || catcherState.Chat[0].Team != "catcher" {
		t.Error("catcher filter failed")
	}

	if len(runnerState.Chat) != 1 || runnerState.Chat[0].Team != "runner" {
		t.Error("runner filter failed")
	}
}

func TestTeamChatFilter_DynamicTeamLookup(t *testing.T) {
	state := NewFilterTestState()

	state.Players = []FilterTestPlayer{
		{ID: "p1", Team: "catcher"},
		{ID: "p2", Team: "runner"},
	}
	state.Chat = []FilterTestMessage{
		{ID: "m1", Team: "catcher", Text: "Catcher only"},
		{ID: "m2", Team: "runner", Text: "Runner only"},
	}

	// Dynamic filter that looks up player's team at filter time
	createDynamicFilter := func(playerID string) FilterFunc[*FilterTestState] {
		return func(s *FilterTestState) *FilterTestState {
			// Find player's team dynamically
			var viewerTeam string
			for _, p := range s.Players {
				if p.ID == playerID {
					viewerTeam = p.Team
					break
				}
			}

			if viewerTeam == "" {
				// Player not found - show nothing
				filtered := s.ShallowClone()
				filtered.Chat = nil
				return filtered
			}

			// Filter by team
			filtered := s.ShallowClone()
			filteredChat := make([]FilterTestMessage, 0)
			for _, msg := range s.Chat {
				if msg.Team == viewerTeam {
					filteredChat = append(filteredChat, msg)
				}
			}
			filtered.Chat = filteredChat
			return filtered
		}
	}

	// Create session with dynamic filters
	tracked := NewTrackedState[*FilterTestState, any](state, nil)
	session := NewTrackedSession[*FilterTestState, any, string](tracked)

	session.Connect("p1", createDynamicFilter("p1"))
	session.Connect("p2", createDynamicFilter("p2"))

	// Test
	p1Filter := session.GetFilter("p1")
	p2Filter := session.GetFilter("p2")

	p1View := p1Filter(state)
	p2View := p2Filter(state)

	if len(p1View.Chat) != 1 || p1View.Chat[0].Team != "catcher" {
		t.Error("p1 (catcher) should only see catcher messages")
	}

	if len(p2View.Chat) != 1 || p2View.Chat[0].Team != "runner" {
		t.Error("p2 (runner) should only see runner messages")
	}
}

func TestTeamChatFilter_EmptyTeam(t *testing.T) {
	state := NewFilterTestState()
	state.Chat = []FilterTestMessage{
		{ID: "m1", Team: "catcher", Text: "Catcher message"},
	}

	// Filter with non-existent team should return nothing
	filter := createTeamChatFilter("spectator")
	filtered := filter(state)

	if len(filtered.Chat) != 0 {
		t.Errorf("spectator should see 0 messages, got %d", len(filtered.Chat))
	}
}

func TestTeamChatFilter_OriginalUnchanged(t *testing.T) {
	state := NewFilterTestState()
	state.Chat = []FilterTestMessage{
		{ID: "m1", Team: "catcher", Text: "Catcher message"},
		{ID: "m2", Team: "runner", Text: "Runner message"},
	}

	originalLen := len(state.Chat)

	// Apply filter
	filter := createTeamChatFilter("catcher")
	filtered := filter(state)

	// Filtered should have 1 message
	if len(filtered.Chat) != 1 {
		t.Errorf("filtered should have 1 message, got %d", len(filtered.Chat))
	}

	// Original should be unchanged
	if len(state.Chat) != originalLen {
		t.Errorf("original state should not be modified, had %d, now %d", originalLen, len(state.Chat))
	}
}

// Test composed filters (multiple filters applied in sequence)
func TestComposedFilters(t *testing.T) {
	state := NewFilterTestState()
	state.Players = []FilterTestPlayer{
		{ID: "p1", Team: "catcher"},
	}
	state.Chat = []FilterTestMessage{
		{ID: "m1", Team: "catcher", Text: "Short"},
		{ID: "m2", Team: "catcher", Text: "This is a longer message"},
		{ID: "m3", Team: "runner", Text: "Runner message"},
	}

	// Filter 1: Team filter
	teamFilter := createTeamChatFilter("catcher")

	// Filter 2: Length filter (only messages > 10 chars)
	lengthFilter := func(s *FilterTestState) *FilterTestState {
		filtered := s.ShallowClone()
		filteredChat := make([]FilterTestMessage, 0)
		for _, msg := range s.Chat {
			if len(msg.Text) > 10 {
				filteredChat = append(filteredChat, msg)
			}
		}
		filtered.Chat = filteredChat
		return filtered
	}

	// Compose filters
	composedFilter := func(s *FilterTestState) *FilterTestState {
		return lengthFilter(teamFilter(s))
	}

	filtered := composedFilter(state)

	// Should only have the long catcher message
	if len(filtered.Chat) != 1 {
		t.Errorf("expected 1 message after composed filter, got %d", len(filtered.Chat))
	}

	if filtered.Chat[0].ID != "m2" {
		t.Errorf("expected message m2, got %s", filtered.Chat[0].ID)
	}
}
