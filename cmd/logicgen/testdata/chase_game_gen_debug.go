//go:build debug
// +build debug

package chase

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/mxkacsa/statesync"
	"github.com/mxkacsa/statesync/debug"
)

var _ = fmt.Sprintf
var _ = math.Sqrt
var _ = rand.Intn
var _ = time.Now

// OnGameTick handles the GameTick event
func OnGameTick(session *statesync.TrackedSession[*ChaseGameState, any, string], dbg debug.DebugHook) error {
	_startTime := time.Now()
	if dbg != nil {
		dbg.OnEventStart(session.ID(), "OnGameTick", nil)
	}

	state := session.State().Get()
	_ = state // may be unused

	// Node: getCurrentTime (GetCurrentTime)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnGameTick", "getCurrentTime", "GetCurrentTime", nil)
	}
	getCurrentTime_timestamp := time.Now().Unix()
	// Node: getGameStart (GetField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnGameTick", "getGameStart", "GetField", nil)
	}
	getGameStart_value := state.GameStartTime()
	// Node: getGameDuration (GetField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnGameTick", "getGameDuration", "GetField", nil)
	}
	getGameDuration_value := state.GameDurationHours()
	// Node: calcDurationSec (Multiply)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnGameTick", "calcDurationSec", "Multiply", nil)
	}
	calcDurationSec_result := getGameDuration_value * 3600
	// Node: calcEndTime (Add)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnGameTick", "calcEndTime", "Add", nil)
	}
	calcEndTime_result := getGameStart_value + calcDurationSec_result
	// Node: checkTimeUp (Compare)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnGameTick", "checkTimeUp", "Compare", nil)
	}
	checkTimeUp_result := getCurrentTime_timestamp >= calcEndTime_result
	// Node: ifTimeUp (If)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnGameTick", "ifTimeUp", "If", nil)
	}
	if checkTimeUp_result {
		// Node: setGameOver (SetField)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnGameTick", "setGameOver", "SetField", nil)
		}
		state.SetIsGameOver(true)
		// Node: setWinner (SetField)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnGameTick", "setWinner", "SetField", nil)
		}
		state.SetWinningTeam("runner")
		// Node: emitGameOver (EmitToAll)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnGameTick", "emitGameOver", "EmitToAll", nil)
		}
		session.Emit("GameOver", nil)
	} else {
	}
	if dbg != nil {
		_duration := float64(time.Since(_startTime).Microseconds()) / 1000.0
		dbg.OnEventEnd(session.ID(), "OnGameTick", _duration, nil)
	}
	return nil
}

// OnTrySpawnBox handles the TrySpawnBox event
func OnTrySpawnBox(session *statesync.TrackedSession[*ChaseGameState, any, string], dbg debug.DebugHook) error {
	_startTime := time.Now()
	if dbg != nil {
		dbg.OnEventStart(session.ID(), "OnTrySpawnBox", nil)
	}

	state := session.State().Get()
	_ = state // may be unused

	// Node: getCurrentTime (GetCurrentTime)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "getCurrentTime", "GetCurrentTime", nil)
	}
	getCurrentTime_timestamp := time.Now().Unix()
	// Node: getNextSpawnTime (GetField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "getNextSpawnTime", "GetField", nil)
	}
	getNextSpawnTime_value := state.NextBoxSpawnTime()
	// Node: getBoxesSpawned (GetField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "getBoxesSpawned", "GetField", nil)
	}
	getBoxesSpawned_value := state.BoxesSpawned()
	// Node: getMaxBoxes (GetField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "getMaxBoxes", "GetField", nil)
	}
	getMaxBoxes_value := state.MaxSurpriseBoxes()
	// Node: checkTimeToSpawn (Compare)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "checkTimeToSpawn", "Compare", nil)
	}
	checkTimeToSpawn_result := getCurrentTime_timestamp >= getNextSpawnTime_value
	// Node: checkCanSpawnMore (Compare)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "checkCanSpawnMore", "Compare", nil)
	}
	checkCanSpawnMore_result := getBoxesSpawned_value < getMaxBoxes_value
	// Node: canSpawn (And)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "canSpawn", "And", nil)
	}
	canSpawn_result := checkTimeToSpawn_result && checkCanSpawnMore_result
	// Node: ifCanSpawn (If)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "ifCanSpawn", "If", nil)
	}
	if canSpawn_result {
		// Node: randomLat (RandomFloat)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "randomLat", "RandomFloat", nil)
		}
		randomLat_result := rand.Float64()*(47.55-47.45) + 47.45
		// Node: randomLng (RandomFloat)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "randomLng", "RandomFloat", nil)
		}
		randomLng_result := rand.Float64()*(19.1-19) + 19
		// Node: randomPoints (RandomInt)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "randomPoints", "RandomInt", nil)
		}
		randomPoints_result := rand.Intn(100-10+1) + 10
		// Node: randomDelay (RandomInt)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "randomDelay", "RandomInt", nil)
		}
		randomDelay_result := rand.Intn(300-60+1) + 60
		// Node: incrementBoxCount (Add)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "incrementBoxCount", "Add", nil)
		}
		incrementBoxCount_result := getBoxesSpawned_value + 1
		// Node: createBox (CreateStruct)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "createBox", "CreateStruct", nil)
		}
		createBox_result := SurpriseBox{
			CollectedBy: "",
			PointValue:  randomPoints_result,
			ID:          incrementBoxCount_result,
			Lat:         randomLat_result,
			Lng:         randomLng_result,
			SpawnTime:   getCurrentTime_timestamp,
			IsCollected: false,
		}
		// Node: appendBox (ArrayAppend)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "appendBox", "ArrayAppend", nil)
		}
		state.AppendSurpriseBoxes(createBox_result)
		// Node: setBoxesSpawned (SetField)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "setBoxesSpawned", "SetField", nil)
		}
		state.SetBoxesSpawned(incrementBoxCount_result)
		// Node: calcNextSpawn (Add)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "calcNextSpawn", "Add", nil)
		}
		calcNextSpawn_result := getCurrentTime_timestamp + randomDelay_result
		// Node: setNextSpawn (SetField)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "setNextSpawn", "SetField", nil)
		}
		state.SetNextBoxSpawnTime(calcNextSpawn_result)
		// Node: emitBoxSpawned (EmitToAll)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnTrySpawnBox", "emitBoxSpawned", "EmitToAll", nil)
		}
		session.Emit("SurpriseBoxSpawned", nil)
	} else {
	}
	if dbg != nil {
		_duration := float64(time.Since(_startTime).Microseconds()) / 1000.0
		dbg.OnEventEnd(session.ID(), "OnTrySpawnBox", _duration, nil)
	}
	return nil
}

// OnPlayerUpdatePosition handles the PlayerUpdatePosition event
func OnPlayerUpdatePosition(session *statesync.TrackedSession[*ChaseGameState, any, string], playerID string, lat float64, lng float64, dbg debug.DebugHook) error {
	_startTime := time.Now()
	if dbg != nil {
		_params := map[string]any{
			"playerID": playerID,
			"lat":      lat,
			"lng":      lng,
		}
		dbg.OnEventStart(session.ID(), "OnPlayerUpdatePosition", _params)
	}

	state := session.State().Get()
	_ = state // may be unused

	// Node: getGameZone (GetField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnPlayerUpdatePosition", "getGameZone", "GetField", nil)
	}
	getGameZone_value := state.GameZone()
	// Node: checkInZone (PointInPolygon)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnPlayerUpdatePosition", "checkInZone", "PointInPolygon", nil)
	}
	// Ray casting algorithm for point-in-polygon
	_checkInZone_n := len(getGameZone_value)
	_checkInZone_inside := false
	for _checkInZone_i, _checkInZone_j := 0, _checkInZone_n-1; _checkInZone_i < _checkInZone_n; _checkInZone_j, _checkInZone_i = _checkInZone_i, _checkInZone_i+1 {
		_checkInZone_xi, _checkInZone_yi := getGameZone_value[_checkInZone_i].Lng, getGameZone_value[_checkInZone_i].Lat
		_checkInZone_xj, _checkInZone_yj := getGameZone_value[_checkInZone_j].Lng, getGameZone_value[_checkInZone_j].Lat
		if ((_checkInZone_yi > lat) != (_checkInZone_yj > lat)) && (lng < (_checkInZone_xj-_checkInZone_xi)*(lat-_checkInZone_yi)/(_checkInZone_yj-_checkInZone_yi)+_checkInZone_xi) {
			_checkInZone_inside = !_checkInZone_inside
		}
	}
	checkInZone_isInside := _checkInZone_inside
	// Node: ifInZone (If)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnPlayerUpdatePosition", "ifInZone", "If", nil)
	}
	if checkInZone_isInside {
		// Node: updatePlayerPos (UpdateWhere)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnPlayerUpdatePosition", "updatePlayerPos", "UpdateWhere", nil)
		}
		updatePlayerPos_count := 0
		for _i := 0; _i < state.PlayersLen(); _i++ {
			_item := state.PlayersAt(_i)
			if _item.ID == playerID {
				_item.Lat = lat
				_item.Lng = lng
				state.UpdatePlayersAt(_i, _item)
				updatePlayerPos_count++
			}
		}
		// Node: emitPosUpdated (EmitToPlayer)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnPlayerUpdatePosition", "emitPosUpdated", "EmitToPlayer", nil)
		}
		session.EmitTo(playerID, "PositionAccepted", nil)
	} else {
		// Node: emitOutOfZone (EmitToPlayer)
		if dbg != nil {
			dbg.OnNodeStart(session.ID(), "OnPlayerUpdatePosition", "emitOutOfZone", "EmitToPlayer", nil)
		}
		session.EmitTo(playerID, "OutOfGameZone", nil)
	}
	if dbg != nil {
		_duration := float64(time.Since(_startTime).Microseconds()) / 1000.0
		dbg.OnEventEnd(session.ID(), "OnPlayerUpdatePosition", _duration, nil)
	}
	return nil
}

// OnTryCollectBox handles the TryCollectBox event
func OnTryCollectBox(session *statesync.TrackedSession[*ChaseGameState, any, string], playerID string, boxID string, dbg debug.DebugHook) error {
	_startTime := time.Now()
	if dbg != nil {
		_params := map[string]any{
			"playerID": playerID,
			"boxID":    boxID,
		}
		dbg.OnEventStart(session.ID(), "OnTryCollectBox", _params)
	}

	state := session.State().Get()
	_ = state // may be unused

	// Node: getPlayer (GetPlayer)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "getPlayer", "GetPlayer", nil)
	}
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
	// Node: getPlayerLat (GetStructField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "getPlayerLat", "GetStructField", nil)
	}
	getPlayerLat_value := getPlayer_player.Lat
	// Node: getPlayerLng (GetStructField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "getPlayerLng", "GetStructField", nil)
	}
	getPlayerLng_value := getPlayer_player.Lng
	// Node: iterateBoxes (ForEachWhere)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "iterateBoxes", "ForEachWhere", nil)
	}
	for iterateBoxes_index := 0; iterateBoxes_index < state.SurpriseBoxesLen(); iterateBoxes_index++ {
		iterateBoxes_item := state.SurpriseBoxesAt(iterateBoxes_index)
		if iterateBoxes_item.ID == boxID {
			// Node: getBoxLat (GetStructField)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "getBoxLat", "GetStructField", nil)
			}
			getBoxLat_value := iterateBoxes_item.Lat
			// Node: getBoxLng (GetStructField)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "getBoxLng", "GetStructField", nil)
			}
			getBoxLng_value := iterateBoxes_item.Lng
			// Node: checkNotCollected (GetStructField)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "checkNotCollected", "GetStructField", nil)
			}
			checkNotCollected_value := iterateBoxes_item.IsCollected
			// Node: isNotCollected (Not)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "isNotCollected", "Not", nil)
			}
			isNotCollected_result := !checkNotCollected_value
			// Node: checkInRange (PointInCircle)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "checkInRange", "PointInCircle", nil)
			}
			// Point in circle check using Haversine distance
			_checkInRange_lat1Rad := getPlayerLat_value * math.Pi / 180.0
			_checkInRange_lat2Rad := getBoxLat_value * math.Pi / 180.0
			_checkInRange_deltaLat := (getBoxLat_value - getPlayerLat_value) * math.Pi / 180.0
			_checkInRange_deltaLng := (getBoxLng_value - getPlayerLng_value) * math.Pi / 180.0
			_checkInRange_a := math.Sin(_checkInRange_deltaLat/2)*math.Sin(_checkInRange_deltaLat/2) + math.Cos(_checkInRange_lat1Rad)*math.Cos(_checkInRange_lat2Rad)*math.Sin(_checkInRange_deltaLng/2)*math.Sin(_checkInRange_deltaLng/2)
			_checkInRange_c := 2 * math.Atan2(math.Sqrt(_checkInRange_a), math.Sqrt(1-_checkInRange_a))
			checkInRange_distance := 6371000.0 * _checkInRange_c
			checkInRange_isInside := checkInRange_distance <= 10
			// Node: canCollect (And)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "canCollect", "And", nil)
			}
			canCollect_result := isNotCollected_result && checkInRange_isInside
			// Node: ifCanCollect (If)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "ifCanCollect", "If", nil)
			}
			if canCollect_result {
				// Node: collectBox (UpdateStruct)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "collectBox", "UpdateStruct", nil)
				}
				iterateBoxes_item.IsCollected = true
				iterateBoxes_item.CollectedBy = playerID
				// Node: emitCollected (EmitToPlayer)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnTryCollectBox", "emitCollected", "EmitToPlayer", nil)
				}
				session.EmitTo(playerID, "BoxCollected", nil)
			}
			state.UpdateSurpriseBoxesAt(iterateBoxes_index, iterateBoxes_item)
		}
	}
	if dbg != nil {
		_duration := float64(time.Since(_startTime).Microseconds()) / 1000.0
		dbg.OnEventEnd(session.ID(), "OnTryCollectBox", _duration, nil)
	}
	return nil
}

// OnCheckCapture handles the CheckCapture event
func OnCheckCapture(session *statesync.TrackedSession[*ChaseGameState, any, string], chaserID string, dbg debug.DebugHook) error {
	_startTime := time.Now()
	if dbg != nil {
		_params := map[string]any{
			"chaserID": chaserID,
		}
		dbg.OnEventStart(session.ID(), "OnCheckCapture", _params)
	}

	state := session.State().Get()
	_ = state // may be unused

	// Node: getChaser (GetPlayer)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnCheckCapture", "getChaser", "GetPlayer", nil)
	}
	var getChaser_player *Player
	for i := range state.Players() {
		p := state.PlayersAt(i)
		if p.ID == chaserID {
			getChaser_player = &p
			break
		}
	}
	if getChaser_player == nil {
		return fmt.Errorf("player not found: %v", chaserID)
	}
	// Node: getChaserLat (GetStructField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnCheckCapture", "getChaserLat", "GetStructField", nil)
	}
	getChaserLat_value := getChaser_player.Lat
	// Node: getChaserLng (GetStructField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnCheckCapture", "getChaserLng", "GetStructField", nil)
	}
	getChaserLng_value := getChaser_player.Lng
	// Node: getCaptureRadius (GetField)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnCheckCapture", "getCaptureRadius", "GetField", nil)
	}
	getCaptureRadius_value := state.CaptureRadius()
	// Node: iterateRunners (ForEachWhere)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnCheckCapture", "iterateRunners", "ForEachWhere", nil)
	}
	for iterateRunners_index := 0; iterateRunners_index < state.PlayersLen(); iterateRunners_index++ {
		iterateRunners_item := state.PlayersAt(iterateRunners_index)
		if iterateRunners_item.Team == "runner" {
			// Node: getRunnerCaptured (GetStructField)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnCheckCapture", "getRunnerCaptured", "GetStructField", nil)
			}
			getRunnerCaptured_value := iterateRunners_item.IsCaptured
			// Node: isNotCaptured (Not)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnCheckCapture", "isNotCaptured", "Not", nil)
			}
			isNotCaptured_result := !getRunnerCaptured_value
			// Node: ifNotCaptured (If)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnCheckCapture", "ifNotCaptured", "If", nil)
			}
			if isNotCaptured_result {
				// Node: getRunnerLat (GetStructField)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnCheckCapture", "getRunnerLat", "GetStructField", nil)
				}
				getRunnerLat_value := iterateRunners_item.Lat
				// Node: getRunnerLng (GetStructField)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnCheckCapture", "getRunnerLng", "GetStructField", nil)
				}
				getRunnerLng_value := iterateRunners_item.Lng
				// Node: checkInCaptureRange (PointInCircle)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnCheckCapture", "checkInCaptureRange", "PointInCircle", nil)
				}
				// Point in circle check using Haversine distance
				_checkInCaptureRange_lat1Rad := getRunnerLat_value * math.Pi / 180.0
				_checkInCaptureRange_lat2Rad := getChaserLat_value * math.Pi / 180.0
				_checkInCaptureRange_deltaLat := (getChaserLat_value - getRunnerLat_value) * math.Pi / 180.0
				_checkInCaptureRange_deltaLng := (getChaserLng_value - getRunnerLng_value) * math.Pi / 180.0
				_checkInCaptureRange_a := math.Sin(_checkInCaptureRange_deltaLat/2)*math.Sin(_checkInCaptureRange_deltaLat/2) + math.Cos(_checkInCaptureRange_lat1Rad)*math.Cos(_checkInCaptureRange_lat2Rad)*math.Sin(_checkInCaptureRange_deltaLng/2)*math.Sin(_checkInCaptureRange_deltaLng/2)
				_checkInCaptureRange_c := 2 * math.Atan2(math.Sqrt(_checkInCaptureRange_a), math.Sqrt(1-_checkInCaptureRange_a))
				checkInCaptureRange_distance := 6371000.0 * _checkInCaptureRange_c
				checkInCaptureRange_isInside := checkInCaptureRange_distance <= getCaptureRadius_value
				// Node: ifInRange (If)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnCheckCapture", "ifInRange", "If", nil)
				}
				if checkInCaptureRange_isInside {
					// Node: captureRunner (UpdateStruct)
					if dbg != nil {
						dbg.OnNodeStart(session.ID(), "OnCheckCapture", "captureRunner", "UpdateStruct", nil)
					}
					iterateRunners_item.IsCaptured = true
					// Node: emitCaptured (EmitToAll)
					if dbg != nil {
						dbg.OnNodeStart(session.ID(), "OnCheckCapture", "emitCaptured", "EmitToAll", nil)
					}
					session.Emit("RunnerCaptured", nil)
				}
			}
			state.UpdatePlayersAt(iterateRunners_index, iterateRunners_item)
		}
	}
	if dbg != nil {
		_duration := float64(time.Since(_startTime).Microseconds()) / 1000.0
		dbg.OnEventEnd(session.ID(), "OnCheckCapture", _duration, nil)
	}
	return nil
}

// OnUpdatePublicPositions handles the UpdatePublicPositions event
func OnUpdatePublicPositions(session *statesync.TrackedSession[*ChaseGameState, any, string], dbg debug.DebugHook) error {
	_startTime := time.Now()
	if dbg != nil {
		dbg.OnEventStart(session.ID(), "OnUpdatePublicPositions", nil)
	}

	state := session.State().Get()
	_ = state // may be unused

	// Node: getCurrentTime (GetCurrentTime)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "getCurrentTime", "GetCurrentTime", nil)
	}
	getCurrentTime_timestamp := time.Now().Unix()
	// Node: iteratePlayers (ForEachWhere)
	if dbg != nil {
		dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "iteratePlayers", "ForEachWhere", nil)
	}
	for iteratePlayers_index := 0; iteratePlayers_index < state.PlayersLen(); iteratePlayers_index++ {
		iteratePlayers_item := state.PlayersAt(iteratePlayers_index)
		if iteratePlayers_item.ID != "" {
			// Node: getLastUpdate (GetStructField)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "getLastUpdate", "GetStructField", nil)
			}
			getLastUpdate_value := iteratePlayers_item.LastPublicUpdate
			// Node: calcTimeSince (Subtract)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "calcTimeSince", "Subtract", nil)
			}
			calcTimeSince_result := getCurrentTime_timestamp - getLastUpdate_value
			// Node: checkNeedsUpdate (Compare)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "checkNeedsUpdate", "Compare", nil)
			}
			checkNeedsUpdate_result := calcTimeSince_result >= 60
			// Node: ifNeedsUpdate (If)
			if dbg != nil {
				dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "ifNeedsUpdate", "If", nil)
			}
			if checkNeedsUpdate_result {
				// Node: getPrivateLat (GetStructField)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "getPrivateLat", "GetStructField", nil)
				}
				getPrivateLat_value := iteratePlayers_item.Lat
				// Node: getPrivateLng (GetStructField)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "getPrivateLng", "GetStructField", nil)
				}
				getPrivateLng_value := iteratePlayers_item.Lng
				// Node: updatePublicPos (UpdateStruct)
				if dbg != nil {
					dbg.OnNodeStart(session.ID(), "OnUpdatePublicPositions", "updatePublicPos", "UpdateStruct", nil)
				}
				iteratePlayers_item.PublicLat = getPrivateLat_value
				iteratePlayers_item.PublicLng = getPrivateLng_value
				iteratePlayers_item.LastPublicUpdate = getCurrentTime_timestamp
			}
			state.UpdatePlayersAt(iteratePlayers_index, iteratePlayers_item)
		}
	}
	if dbg != nil {
		_duration := float64(time.Since(_startTime).Microseconds()) / 1000.0
		dbg.OnEventEnd(session.ID(), "OnUpdatePublicPositions", _duration, nil)
	}
	return nil
}
