package eval

import (
	"testing"
	"time"

	"github.com/mxkacsa/statesync/cmd/logicgen/ast"
)

// Test entities for permission testing
type PermTestPlayer struct {
	ID       string
	Name     string
	Score    int64
	Hand     []int32 // Owner only
	Position PermTestPosition
}

type PermTestPosition struct {
	X float64
	Y float64
}

type PermTestDrone struct {
	ID      string
	OwnerID string // Different field name for owner
	Health  int32
	X       float64
	Y       float64
}

type PermTestGameState struct {
	Players    []PermTestPlayer
	Drones     []PermTestDrone
	ServerSeed int64 // Server only
	Phase      string
}

// createTestPermissionSchema creates a permission schema for testing
func createTestPermissionSchema() *PermissionSchema {
	schema := NewPermissionSchema()

	// Player type - ID is the owner field
	playerType := schema.RegisterType("PermTestPlayer", "ID")
	playerType.SetFieldPermission("Score", WriteServer) // Only server can modify score
	playerType.SetFieldPermission("Hand", WriteOwner)   // Only owner can modify hand
	playerType.SetFieldPermission("Name", WriteOwner)   // Only owner can modify name
	// Position has no restriction (WriteAnyone)

	// Drone type - OwnerID is the owner field
	droneType := schema.RegisterType("PermTestDrone", "OwnerID")
	droneType.SetFieldPermission("Health", WriteServer) // Only server can modify health
	droneType.SetFieldPermission("X", WriteOwner)       // Only owner can modify position
	droneType.SetFieldPermission("Y", WriteOwner)       // Only owner can modify position

	// GameState - root state has server-only fields
	gameStateType := schema.RegisterType("PermTestGameState", "")
	gameStateType.SetFieldPermission("ServerSeed", WriteServer) // Only server
	gameStateType.SetFieldPermission("Phase", WriteServer)      // Only server

	return schema
}

func TestPermissionChecker_ServerCanWriteAnything(t *testing.T) {
	schema := createTestPermissionSchema()
	checker := NewPermissionChecker(schema) // Server mode by default

	player := &PermTestPlayer{ID: "player1", Name: "Alice", Score: 100}

	// Server should be able to write all fields
	tests := []string{"Score", "Hand", "Name", "Position"}
	for _, field := range tests {
		err := checker.CanWrite(player, field)
		if err != nil {
			t.Errorf("Server should be able to write %s, got error: %v", field, err)
		}
	}
}

func TestPermissionChecker_PlayerCannotWriteServerOnlyFields(t *testing.T) {
	schema := createTestPermissionSchema()
	checker := NewPermissionChecker(schema).WithSender("player1")

	player := &PermTestPlayer{ID: "player1", Name: "Alice", Score: 100}

	// Player should NOT be able to write Score (server only)
	err := checker.CanWrite(player, "Score")
	if err == nil {
		t.Error("Player should not be able to write Score (server only)")
	}
	if !IsPermissionError(err) {
		t.Errorf("Expected PermissionError, got: %T", err)
	}

	permErr := err.(*PermissionError)
	if permErr.Required != WriteServer {
		t.Errorf("Expected Required=WriteServer, got: %v", permErr.Required)
	}
}

func TestPermissionChecker_OwnerCanWriteOwnerFields(t *testing.T) {
	schema := createTestPermissionSchema()

	player := &PermTestPlayer{ID: "player1", Name: "Alice", Score: 100}

	// player1 (the owner) should be able to write Hand
	checkerOwner := NewPermissionChecker(schema).WithSender("player1")
	err := checkerOwner.CanWrite(player, "Hand")
	if err != nil {
		t.Errorf("Owner should be able to write Hand, got error: %v", err)
	}

	err = checkerOwner.CanWrite(player, "Name")
	if err != nil {
		t.Errorf("Owner should be able to write Name, got error: %v", err)
	}
}

func TestPermissionChecker_NonOwnerCannotWriteOwnerFields(t *testing.T) {
	schema := createTestPermissionSchema()

	player := &PermTestPlayer{ID: "player1", Name: "Alice", Score: 100}

	// player2 (NOT the owner) should NOT be able to write Hand
	checkerOther := NewPermissionChecker(schema).WithSender("player2")
	err := checkerOther.CanWrite(player, "Hand")
	if err == nil {
		t.Error("Non-owner should not be able to write Hand")
	}
	if !IsPermissionError(err) {
		t.Errorf("Expected PermissionError, got: %T", err)
	}

	permErr := err.(*PermissionError)
	if permErr.Required != WriteOwner {
		t.Errorf("Expected Required=WriteOwner, got: %v", permErr.Required)
	}
	if permErr.OwnerID != "player1" {
		t.Errorf("Expected OwnerID=player1, got: %v", permErr.OwnerID)
	}
	if permErr.SenderID != "player2" {
		t.Errorf("Expected SenderID=player2, got: %v", permErr.SenderID)
	}
}

func TestPermissionChecker_AnyoneCanWriteUnrestrictedFields(t *testing.T) {
	schema := createTestPermissionSchema()

	player := &PermTestPlayer{ID: "player1", Name: "Alice", Score: 100}

	// Any player should be able to write Position (no restriction)
	checkerOther := NewPermissionChecker(schema).WithSender("player2")
	err := checkerOther.CanWrite(player, "Position")
	if err != nil {
		t.Errorf("Anyone should be able to write Position, got error: %v", err)
	}
}

func TestPermissionChecker_DifferentOwnerField(t *testing.T) {
	schema := createTestPermissionSchema()

	drone := &PermTestDrone{ID: "drone1", OwnerID: "player1", Health: 100}

	// player1 (owner via OwnerID) should be able to write X
	checkerOwner := NewPermissionChecker(schema).WithSender("player1")
	err := checkerOwner.CanWrite(drone, "X")
	if err != nil {
		t.Errorf("Owner should be able to write X, got error: %v", err)
	}

	// player2 should NOT be able to write X
	checkerOther := NewPermissionChecker(schema).WithSender("player2")
	err = checkerOther.CanWrite(drone, "X")
	if err == nil {
		t.Error("Non-owner should not be able to write X on drone")
	}

	// But no one except server can write Health
	err = checkerOwner.CanWrite(drone, "Health")
	if err == nil {
		t.Error("Even owner should not be able to write Health (server only)")
	}
}

func TestPermissionChecker_UnregisteredTypeHasNoRestrictions(t *testing.T) {
	schema := createTestPermissionSchema()

	// Unknown type should have no restrictions
	unknownEntity := struct{ Value int }{Value: 42}

	checker := NewPermissionChecker(schema).WithSender("player1")
	err := checker.CanWrite(unknownEntity, "Value")
	if err != nil {
		t.Errorf("Unknown type should have no restrictions, got error: %v", err)
	}
}

func TestPermissionChecker_NilSchemaHasNoRestrictions(t *testing.T) {
	checker := NewPermissionChecker(nil)

	player := &PermTestPlayer{ID: "player1", Name: "Alice", Score: 100}

	err := checker.CanWrite(player, "Score")
	if err != nil {
		t.Errorf("Nil schema should have no restrictions, got error: %v", err)
	}
}

// Integration tests with Context.SetPath

func TestContext_SetPath_WithPermissions_ServerCanSetAll(t *testing.T) {
	state := &PermTestGameState{
		Players: []PermTestPlayer{
			{ID: "player1", Name: "Alice", Score: 100},
		},
	}

	schema := createTestPermissionSchema()
	ctx := NewContext(state, time.Millisecond*16, 1).WithPermissions(schema)

	// Server (no sender) should be able to set Score
	// Note: Use WithEntity for slice elements because Go reflection can't set on slice copies
	entityCtx := ctx.WithEntity(&state.Players[0], 0)
	err := entityCtx.SetPath("$.Score", int64(200))
	if err != nil {
		t.Errorf("Server should be able to set Score, got error: %v", err)
	}

	if state.Players[0].Score != 200 {
		t.Errorf("Expected Score=200, got: %d", state.Players[0].Score)
	}
}

func TestContext_SetPath_WithPermissions_PlayerCannotSetServerOnly(t *testing.T) {
	state := &PermTestGameState{
		Players: []PermTestPlayer{
			{ID: "player1", Name: "Alice", Score: 100},
		},
	}

	schema := createTestPermissionSchema()
	ctx := NewContext(state, time.Millisecond*16, 1).
		WithPermissions(schema).
		WithSender("player1")

	// Player should NOT be able to set Score (server only)
	// First navigate to the player entity
	playerCtx := ctx.WithEntity(&state.Players[0], 0)
	err := playerCtx.SetPath("$.Score", int64(9999))
	if err == nil {
		t.Error("Player should not be able to set Score (server only)")
	}
	if !IsPermissionError(err) {
		t.Errorf("Expected PermissionError, got: %T - %v", err, err)
	}

	// Score should remain unchanged
	if state.Players[0].Score != 100 {
		t.Errorf("Score should not have changed, got: %d", state.Players[0].Score)
	}
}

func TestContext_SetPath_WithPermissions_OwnerCanSetOwnField(t *testing.T) {
	state := &PermTestGameState{
		Players: []PermTestPlayer{
			{ID: "player1", Name: "Alice", Score: 100},
		},
	}

	schema := createTestPermissionSchema()
	ctx := NewContext(state, time.Millisecond*16, 1).
		WithPermissions(schema).
		WithSender("player1")

	// Player1 should be able to set their own Name
	playerCtx := ctx.WithEntity(&state.Players[0], 0)
	err := playerCtx.SetPath("$.Name", "Bob")
	if err != nil {
		t.Errorf("Owner should be able to set Name, got error: %v", err)
	}

	if state.Players[0].Name != "Bob" {
		t.Errorf("Expected Name=Bob, got: %s", state.Players[0].Name)
	}
}

func TestContext_SetPath_WithPermissions_OtherPlayerCannotSetOwnerField(t *testing.T) {
	state := &PermTestGameState{
		Players: []PermTestPlayer{
			{ID: "player1", Name: "Alice", Score: 100},
		},
	}

	schema := createTestPermissionSchema()
	ctx := NewContext(state, time.Millisecond*16, 1).
		WithPermissions(schema).
		WithSender("player2") // Different player!

	// Player2 should NOT be able to set player1's Name
	playerCtx := ctx.WithEntity(&state.Players[0], 0)
	err := playerCtx.SetPath("$.Name", "Hacked")
	if err == nil {
		t.Error("Other player should not be able to set owner's Name")
	}
	if !IsPermissionError(err) {
		t.Errorf("Expected PermissionError, got: %T - %v", err, err)
	}

	// Name should remain unchanged
	if state.Players[0].Name != "Alice" {
		t.Errorf("Name should not have changed, got: %s", state.Players[0].Name)
	}
}

func TestContext_SetPath_NoPermissionSchema_NoRestrictions(t *testing.T) {
	state := &PermTestGameState{
		Players: []PermTestPlayer{
			{ID: "player1", Name: "Alice", Score: 100},
		},
	}

	// No permission schema = no restrictions
	ctx := NewContext(state, time.Millisecond*16, 1).WithSender("player2")

	// Any player should be able to set anything
	playerCtx := ctx.WithEntity(&state.Players[0], 0)
	err := playerCtx.SetPath("$.Score", int64(9999))
	if err != nil {
		t.Errorf("No schema means no restrictions, got error: %v", err)
	}

	if state.Players[0].Score != 9999 {
		t.Errorf("Expected Score=9999, got: %d", state.Players[0].Score)
	}
}

// Test with event-triggered context
func TestContext_WithEvent_SetsPermissionChecker(t *testing.T) {
	state := &PermTestGameState{
		Players: []PermTestPlayer{
			{ID: "player1", Name: "Alice", Score: 100},
		},
	}

	schema := createTestPermissionSchema()
	ctx := NewContext(state, time.Millisecond*16, 1).WithPermissions(schema)

	// Create an event from player2
	event := &ast.Event{
		Name:   "TestEvent",
		Sender: "player2",
	}

	eventCtx := ctx.WithEvent(event)

	// Verify sender is set
	if eventCtx.SenderID != "player2" {
		t.Errorf("Expected SenderID=player2, got: %s", eventCtx.SenderID)
	}

	// player2 should not be able to modify player1's owner-only fields
	playerCtx := eventCtx.WithEntity(&state.Players[0], 0)
	err := playerCtx.SetPath("$.Name", "Hacked")
	if err == nil {
		t.Error("Player2 (via event) should not be able to set player1's Name")
	}
}

// Test Spawn effect owner assignment
func TestSpawnEffect_OwnerAssignment(t *testing.T) {
	state := &PermTestGameState{
		Drones: []PermTestDrone{},
	}

	schema := createTestPermissionSchema()

	// Test 1: When sender spawns entity, owner should be set to sender
	t.Run("DefaultOwnerFromSender", func(t *testing.T) {
		ctx := NewContext(state, time.Millisecond*16, 1).
			WithPermissions(schema).
			WithSender("player1")

		effect := &ast.Effect{
			Type:   ast.EffectTypeSpawn,
			Entity: "PermTestDrone",
			Fields: map[string]interface{}{
				"ID":     "drone1",
				"Health": 100,
			},
		}

		ee := NewEffectEvaluator()
		err := ee.Apply(ctx, effect, nil)
		if err != nil {
			t.Errorf("Spawn should succeed: %v", err)
		}
		// Note: Actual entity creation is TODO in the implementation
		// This test validates the logic flow without actual entity creation
	})

	// Test 2: Explicit owner in fields should not be overwritten
	t.Run("ExplicitOwnerNotOverwritten", func(t *testing.T) {
		ctx := NewContext(state, time.Millisecond*16, 1).
			WithPermissions(schema).
			WithSender("player1")

		effect := &ast.Effect{
			Type:   ast.EffectTypeSpawn,
			Entity: "PermTestDrone",
			Fields: map[string]interface{}{
				"ID":      "drone2",
				"OwnerID": "player2", // Explicit owner
				"Health":  100,
			},
		}

		ee := NewEffectEvaluator()
		err := ee.Apply(ctx, effect, nil)
		if err != nil {
			t.Errorf("Spawn should succeed: %v", err)
		}
	})

	// Test 3: Server spawn without sender should not set owner automatically
	t.Run("ServerSpawnNoAutoOwner", func(t *testing.T) {
		ctx := NewContext(state, time.Millisecond*16, 1).
			WithPermissions(schema)
		// No sender = server mode

		effect := &ast.Effect{
			Type:   ast.EffectTypeSpawn,
			Entity: "PermTestDrone",
			Fields: map[string]interface{}{
				"ID":     "drone3",
				"Health": 100,
			},
		}

		ee := NewEffectEvaluator()
		err := ee.Apply(ctx, effect, nil)
		if err != nil {
			t.Errorf("Spawn should succeed: %v", err)
		}
	})
}

// Test realistic game scenario
func TestPermissions_RealisticScenario_CatchGame(t *testing.T) {
	// Scenario: A catch game where players can only move their own position,
	// but only the server can modify scores

	type CatchPlayer struct {
		ID       string
		Name     string
		X        float64
		Y        float64
		Score    int64
		IsCaught bool
	}

	type CatchGameState struct {
		Players []CatchPlayer
		Round   int32
	}

	state := &CatchGameState{
		Players: []CatchPlayer{
			{ID: "alice", Name: "Alice", X: 0, Y: 0, Score: 0},
			{ID: "bob", Name: "Bob", X: 10, Y: 10, Score: 0},
		},
		Round: 1,
	}

	schema := NewPermissionSchema()
	playerType := schema.RegisterType("CatchPlayer", "ID")
	playerType.SetFieldPermission("X", WriteOwner)         // Owner can move
	playerType.SetFieldPermission("Y", WriteOwner)         // Owner can move
	playerType.SetFieldPermission("Score", WriteServer)    // Only server awards points
	playerType.SetFieldPermission("IsCaught", WriteServer) // Only server marks caught

	gameType := schema.RegisterType("CatchGameState", "")
	gameType.SetFieldPermission("Round", WriteServer) // Only server advances round

	// Test 1: Alice can move herself
	ctx := NewContext(state, time.Millisecond*16, 1).
		WithPermissions(schema).
		WithSender("alice")

	aliceCtx := ctx.WithEntity(&state.Players[0], 0)
	if err := aliceCtx.SetPath("$.X", 5.0); err != nil {
		t.Errorf("Alice should be able to move herself: %v", err)
	}
	if state.Players[0].X != 5.0 {
		t.Errorf("Expected X=5.0, got: %f", state.Players[0].X)
	}

	// Test 2: Bob cannot move Alice
	bobCtx := ctx.WithSender("bob").WithEntity(&state.Players[0], 0)
	if err := bobCtx.SetPath("$.X", 99.0); err == nil {
		t.Error("Bob should not be able to move Alice")
	}
	if state.Players[0].X != 5.0 {
		t.Errorf("Alice's X should not have changed, got: %f", state.Players[0].X)
	}

	// Test 3: No player can modify scores
	if err := aliceCtx.SetPath("$.Score", int64(100)); err == nil {
		t.Error("Alice should not be able to modify her own score")
	}

	// Test 4: Server can modify scores
	serverCtx := NewContext(state, time.Millisecond*16, 1).
		WithPermissions(schema).
		WithEntity(&state.Players[0], 0)
	if err := serverCtx.SetPath("$.Score", int64(100)); err != nil {
		t.Errorf("Server should be able to set score: %v", err)
	}
	if state.Players[0].Score != 100 {
		t.Errorf("Expected Score=100, got: %d", state.Players[0].Score)
	}

	// Test 5: Server can mark player as caught
	if err := serverCtx.SetPath("$.IsCaught", true); err != nil {
		t.Errorf("Server should be able to mark caught: %v", err)
	}
	if !state.Players[0].IsCaught {
		t.Error("Expected IsCaught=true")
	}
}
