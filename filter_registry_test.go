package statesync

import (
	"testing"
)

type RegistryTestState struct {
	Score int
	Name  string
}

func TestFilterRegistry_AddRemove(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	// Add a filter
	filter := func(s *RegistryTestState) *RegistryTestState {
		return &RegistryTestState{Score: s.Score * 2, Name: s.Name}
	}

	registry.Add("player1", "doubleScore", filter)

	// Check it exists
	if !registry.Has("player1", "doubleScore") {
		t.Error("filter should exist")
	}

	// Check count
	if registry.Count("player1") != 1 {
		t.Errorf("expected 1 filter, got %d", registry.Count("player1"))
	}

	// Remove it
	if !registry.Remove("player1", "doubleScore") {
		t.Error("remove should return true")
	}

	// Check it's gone
	if registry.Has("player1", "doubleScore") {
		t.Error("filter should be removed")
	}

	// Remove again should return false
	if registry.Remove("player1", "doubleScore") {
		t.Error("second remove should return false")
	}
}

func TestFilterRegistry_Compose(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	// Add multiple filters
	registry.Add("player1", "doubleScore", func(s *RegistryTestState) *RegistryTestState {
		return &RegistryTestState{Score: s.Score * 2, Name: s.Name}
	})

	registry.Add("player1", "addTen", func(s *RegistryTestState) *RegistryTestState {
		return &RegistryTestState{Score: s.Score + 10, Name: s.Name}
	})

	// Compose filters
	composed := registry.Compose("player1")
	if composed == nil {
		t.Fatal("composed filter should not be nil")
	}

	// Apply composed filter
	state := &RegistryTestState{Score: 5, Name: "Test"}
	result := composed(state)

	// Order is not guaranteed, but both should be applied
	// If doubleScore first: 5*2=10, then +10 = 20
	// If addTen first: 5+10=15, then *2 = 30
	if result.Score != 20 && result.Score != 30 {
		t.Errorf("unexpected score: %d (expected 20 or 30)", result.Score)
	}
}

func TestFilterRegistry_ComposeSingle(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	registry.Add("player1", "double", func(s *RegistryTestState) *RegistryTestState {
		return &RegistryTestState{Score: s.Score * 2, Name: s.Name}
	})

	composed := registry.Compose("player1")
	if composed == nil {
		t.Fatal("composed filter should not be nil")
	}

	state := &RegistryTestState{Score: 7, Name: "Test"}
	result := composed(state)

	if result.Score != 14 {
		t.Errorf("expected 14, got %d", result.Score)
	}
}

func TestFilterRegistry_ComposeEmpty(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	composed := registry.Compose("nonexistent")
	if composed != nil {
		t.Error("composed filter should be nil for unknown viewer")
	}
}

func TestFilterRegistry_ComposeWith(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	registry.Add("player1", "addFive", func(s *RegistryTestState) *RegistryTestState {
		return &RegistryTestState{Score: s.Score + 5, Name: s.Name}
	})

	baseFilter := func(s *RegistryTestState) *RegistryTestState {
		return &RegistryTestState{Score: s.Score * 3, Name: s.Name}
	}

	composed := registry.ComposeWith("player1", baseFilter)

	state := &RegistryTestState{Score: 2, Name: "Test"}
	result := composed(state)

	// Base first: 2*3=6, then registry: 6+5=11
	if result.Score != 11 {
		t.Errorf("expected 11, got %d", result.Score)
	}
}

func TestFilterRegistry_GetAll(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	registry.Add("player1", "filter1", func(s *RegistryTestState) *RegistryTestState { return s })
	registry.Add("player1", "filter2", func(s *RegistryTestState) *RegistryTestState { return s })
	registry.Add("player2", "filter3", func(s *RegistryTestState) *RegistryTestState { return s })

	ids := registry.GetAll("player1")
	if len(ids) != 2 {
		t.Errorf("expected 2 filters for player1, got %d", len(ids))
	}

	ids2 := registry.GetAll("player2")
	if len(ids2) != 1 {
		t.Errorf("expected 1 filter for player2, got %d", len(ids2))
	}
}

func TestFilterRegistry_Clear(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	registry.Add("player1", "filter1", func(s *RegistryTestState) *RegistryTestState { return s })
	registry.Add("player1", "filter2", func(s *RegistryTestState) *RegistryTestState { return s })

	registry.Clear("player1")

	if registry.Count("player1") != 0 {
		t.Error("expected 0 filters after clear")
	}
}

func TestFilterRegistry_ClearAll(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	registry.Add("player1", "filter1", func(s *RegistryTestState) *RegistryTestState { return s })
	registry.Add("player2", "filter2", func(s *RegistryTestState) *RegistryTestState { return s })

	registry.ClearAll()

	if registry.Count("player1") != 0 || registry.Count("player2") != 0 {
		t.Error("expected 0 filters after clearAll")
	}
}

func TestFilterRegistry_Get(t *testing.T) {
	registry := NewFilterRegistry[*RegistryTestState, string]()

	originalFilter := func(s *RegistryTestState) *RegistryTestState {
		return &RegistryTestState{Score: 999, Name: s.Name}
	}

	registry.Add("player1", "myFilter", originalFilter)

	// Get existing filter
	filter := registry.Get("player1", "myFilter")
	if filter == nil {
		t.Fatal("filter should not be nil")
	}

	result := filter(&RegistryTestState{Score: 1})
	if result.Score != 999 {
		t.Errorf("expected 999, got %d", result.Score)
	}

	// Get non-existing filter
	filter = registry.Get("player1", "nonexistent")
	if filter != nil {
		t.Error("non-existing filter should be nil")
	}
}
