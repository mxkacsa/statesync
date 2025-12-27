package main

import (
	"fmt"

	"github.com/mxkacsa/statesync"
)

// OnScoreUpdate handles the ScoreUpdate event
func OnScoreUpdate(session *statesync.TrackedSession[*GameState, any, string], playerID string, points int64) error {
	// Node: getPlayer (GetPlayer)
	state := session.State().Get()
	playerIndex := -1
	for i := 0; i < state.PlayersLen(); i++ {
		player := state.PlayersAt(i)
		if player.ID() == playerID {
			playerIndex = i
			break
		}
	}
	if playerIndex < 0 {
		return fmt.Errorf("player not found: %s", playerID)
	}

	// Node: addPoints (Add)
	player := state.PlayersAt(playerIndex)
	newScore := player.Score() + points
	player.SetScore(newScore)
	state.UpdatePlayersAt(playerIndex, player)

	// Node: notify (EmitToAll)
	enc := statesync.NewEventPayloadEncoder()
	session.Emit("ScoreUpdated", enc.Bytes())
	return nil
}
