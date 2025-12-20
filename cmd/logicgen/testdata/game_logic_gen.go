package game

import (
	"fmt"
	"math"

	"github.com/mxkacsa/statesync"
)

var _ = fmt.Sprintf
var _ = math.Sqrt

// OnCardPlayed handles the CardPlayed event
func OnCardPlayed(session *statesync.TrackedSession[*GameState, any, string], playerID string, cardIndex int) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getPlayer (GetPlayer)
	var getPlayer_player *Player
	for i := range state.Players() {
		p := state.PlayersAt(i)
		if p.ID == playerID {
			getPlayer_player = &p
			break
		}
	}
	if getPlayer_player == nil {
		return fmt.Errorf("player not found: %v", playerID)
	}
	// Node: addScore (Add)
	addScore_result := 100 + 10
	// Node: emitCardPlayed (EmitToAll)
	session.Emit("CardPlayedEvent", nil)
	return nil
}

// OnSetTeamTarget handles the SetTeamTarget event
func OnSetTeamTarget(session *statesync.TrackedSession[*GameState, any, string], teamID int32, targetX float32, targetY float32) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: updateTeamPositions (UpdateWhere)
	updateTeamPositions_count := 0
	for _i := 0; _i < state.PlayersLen(); _i++ {
		_item := state.PlayersAt(_i)
		if _item.Team == teamID {
			_item.TargetX = targetX
			state.UpdatePlayersAt(_i, _item)
			updateTeamPositions_count++
		}
	}
	// Node: emitTeamMoved (EmitToAll)
	session.Emit("TeamMoved", nil)
	return nil
}

// OnNextRound handles the NextRound event
func OnNextRound(session *statesync.TrackedSession[*GameState, any, string]) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getRound (GetField)
	getRound_value := state.Round()
	// Node: incrementRound (Add)
	incrementRound_result := getRound_value + 1
	// Node: setRound (SetField)
	state.SetRound(incrementRound_result)
	// Node: countReadyPlayers (CountWhere)
	countReadyPlayers_count := 0
	for _, _item := range state.Players() {
		if _item.IsReady == true {
			countReadyPlayers_count++
		}
	}
	// Node: emitRoundStarted (EmitToAll)
	session.Emit("RoundStarted", nil)
	return nil
}

// OnPlayerConnect handles the PlayerConnect event
func OnPlayerConnect(session *statesync.TrackedSession[*GameState, any, string], playerID string, playerName string) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: checkPlayerCount (ArrayLength)
	checkPlayerCount_length := len(state.Players())
	// Node: checkLimit (Compare)
	checkLimit_result := checkPlayerCount_length < 4
	// Node: ifCanJoin (If)
	if checkLimit_result {
		// Node: emitJoined (EmitToPlayer)
		session.EmitTo(playerID, "JoinAccepted", nil)
	} else {
		// Node: emitRejected (EmitToPlayer)
		session.EmitTo(playerID, "JoinRejected", nil)
	}
	return nil
}
