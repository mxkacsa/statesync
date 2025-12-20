package example

import (
	"fmt"
	"github.com/mxkacsa/statesync"
)

// OnScoreUpdate handles the ScoreUpdate event
func OnScoreUpdate(session *statesync.TrackedSession[*GameState, any, string], playerID string, points int64) error {
	// Node: getPlayer (GetPlayer)
	state := session.State().Get()
	var getPlayer_player *Player
	for i := range state.Players {
		if state.Players[i].ID == playerID {
			getPlayer_player = &state.Players[i]
			break
		}
	}
	if getPlayer_player == nil {
		return fmt.Errorf("player not found: %s", playerID)
	}
	// Node: addPoints (Add)
	addPoints_result := getPlayer_player + points
	// Node: notify (EmitToAll)
	enc := statesync.NewEventPayloadEncoder()
	session.Emit("ScoreUpdated", enc.Bytes())
	return nil
}
