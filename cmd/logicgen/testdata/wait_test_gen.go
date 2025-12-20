package waitest

import (
	"context"
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

// OnAttackWithCooldown handles the AttackWithCooldown event
func OnAttackWithCooldown(ctx context.Context, session *statesync.TrackedSession[*WaitTestState, any, string], playerID string, damage int32) error {
	state := session.State().Get()
	_ = state // may be unused
	_ = ctx   // may be unused

	// Node: getDamage (Constant)
	getDamage_value := damage
	// Node: waitCooldown (Wait)
	select {
	case <-time.After(time.Duration(500) * time.Millisecond):
		// Wait completed
	case <-ctx.Done():
		return ctx.Err()
	}
	// Node: applyDamage (SetField)
	state.SetLastDamage(damage)
	return nil
}

// OnWaitForCondition handles the WaitForCondition event
func OnWaitForCondition(ctx context.Context, session *statesync.TrackedSession[*WaitTestState, any, string]) error {
	state := session.State().Get()
	_ = state // may be unused
	_ = ctx   // may be unused

	// Node: getReady (GetField)
	getReady_value := state.IsReady()
	// Node: waitUntilReady (WaitUntil)
	_waitUntilReady_deadline := time.Now().Add(time.Duration(5000) * time.Millisecond)
	_waitUntilReady_ticker := time.NewTicker(time.Duration(100) * time.Millisecond)
	defer _waitUntilReady_ticker.Stop()
	waitUntilReady_timedOut := false

_waitUntilReady_loop:
	for {
		if getReady_value {
			break _waitUntilReady_loop
		}
		if 5000 > 0 && time.Now().After(_waitUntilReady_deadline) {
			waitUntilReady_timedOut = true
			break _waitUntilReady_loop
		}
		select {
		case <-_waitUntilReady_ticker.C:
			// Check again
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	// Node: checkTimedOut (If)
	if waitUntilReady_timedOut {
		// Node: setFailed (SetField)
		state.SetStatus("timeout")
	} else {
		// Node: setSuccess (SetField)
		state.SetStatus("ready")
	}
	return nil
}
