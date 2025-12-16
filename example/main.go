package main

import (
	"fmt"

	"github.com/mxkacsa/statesync"
)

func main() {
	fmt.Println("=== StateSync DSL Example ===")
	fmt.Println()

	// Create state using generated types from game.schema
	game := NewGameState()
	tracked := statesync.NewTrackedState[*GameState, string](game, nil)
	session := statesync.NewTrackedSession[*GameState, string, string](tracked)

	// Connect clients
	session.Connect("alice", nil)
	session.Connect("bob", nil)

	// Set initial state
	game.SetRound(1)
	game.SetPhase("lobby")

	// Add players using generated methods
	player1 := NewPlayer()
	player1.SetID("alice")
	player1.SetName("Alice")
	player1.SetScore(0)
	player1.SetHand([]int32{1, 2, 3})
	player1.SetReady(true)

	player2 := NewPlayer()
	player2.SetID("bob")
	player2.SetName("Bob")
	player2.SetScore(0)
	player2.SetHand([]int32{4, 5, 6})
	player2.SetReady(false)

	// Note: In real usage, Players would be tracked nested objects
	// For now, demonstrate basic tracking

	fmt.Println("--- Initial state broadcast ---")
	diffs := session.Tick()
	for id, data := range diffs {
		fmt.Printf("%s: %d bytes (full state)\n", id, len(data))
	}

	// Update game state
	fmt.Println("\n--- Game starts ---")
	game.SetPhase("playing")
	game.SetRound(2)

	diffs = session.Tick()
	for id, data := range diffs {
		fmt.Printf("%s: %d bytes (delta)\n", id, len(data))
	}

	// Single field update
	fmt.Println("\n--- Score update ---")
	game.SetScoresKey("alice", 100)
	game.SetScoresKey("bob", 50)

	diffs = session.Tick()
	for id, data := range diffs {
		fmt.Printf("%s: %d bytes (scores)\n", id, len(data))
	}

	fmt.Println("\n=== Done ===")
	fmt.Println()
	fmt.Println("Generated types from game.schema:")
	fmt.Printf("- Player: ID=%s, Name=%s, Score=%d\n",
		player1.ID(), player1.Name(), player1.Score())
	fmt.Printf("- GameState: Round=%d, Phase=%s\n",
		game.Round(), game.Phase())
}
