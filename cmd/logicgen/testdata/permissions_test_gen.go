package generated

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
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

var _OnAdminCommand_allowedPlayers = map[string]bool{
	"admin1": true,
	"admin2": true,
}

// OnStartGame handles the StartGame event
func OnStartGame(session *statesync.TrackedSession[*GameState, any, string], senderID string) error {
	// Permission: hostOnly
	if !session.IsHost(senderID) {
		return ErrNotHost
	}

	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: setStarted (SetField)
	state.SetGameStarted(true)
	// Node: emitStarted (EmitToAll)
	session.Emit("GameStarted", nil)
	return nil
}

// OnPlayerMove handles the PlayerMove event
func OnPlayerMove(session *statesync.TrackedSession[*GameState, any, string], senderID string, playerID string, x float64, y float64) error {
	// Permission: playerParam must match sender
	if playerID != senderID {
		return ErrNotAllowed
	}

	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: logMove (EmitToAll)
	session.Emit("PlayerMoved", nil)
	return nil
}

// OnAdminCommand handles the AdminCommand event
func OnAdminCommand(session *statesync.TrackedSession[*GameState, any, string], senderID string, command string) error {
	// Permission: allowedPlayers
	if !_OnAdminCommand_allowedPlayers[senderID] {
		return ErrNotAllowed
	}

	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: logAdmin (EmitToAll)
	session.Emit("AdminAction", nil)
	return nil
}

// OnKickPlayer handles the KickPlayer event
func OnKickPlayer(session *statesync.TrackedSession[*GameState, any, string], senderID string, targetPlayerID string, reason string) error {
	// Permission: hostOnly
	if !session.IsHost(senderID) {
		return ErrNotHost
	}

	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: checkNotSelf (Compare)
	checkNotSelf_result := targetPlayerID != senderID
	// Node: ifNotSelf (If)
	if checkNotSelf_result {
		// Node: doKick (KickPlayer)
		doKick_kicked := session.Kick(targetPlayerID, reason)
	} else {
		// Node: emitKickFailed (EmitToPlayer)
		session.EmitTo(senderID, "KickFailed", "Cannot kick yourself")
	}
	return nil
}

// OnGetHost handles the GetHost event
func OnGetHost(session *statesync.TrackedSession[*GameState, any, string], senderID string) error {
	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: getHost (GetHostPlayer)
	getHost_hostPlayerID := session.HostPlayerID()
	// Node: emitHost (EmitToAll)
	session.Emit("CurrentHost", nil)
	return nil
}

// OnCheckHost handles the CheckHost event
func OnCheckHost(session *statesync.TrackedSession[*GameState, any, string], senderID string, playerID string) error {
	state := session.State().Get()
	_ = state    // may be unused
	_ = senderID // may be unused

	// Node: checkHost (IsHost)
	checkHost_isHost := session.IsHost(playerID)
	// Node: emitResult (EmitToAll)
	session.Emit("HostCheckResult", nil)
	return nil
}
