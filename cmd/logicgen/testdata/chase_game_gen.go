package chase

import (
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

// OnGameTick handles the GameTick event
func OnGameTick(session *statesync.TrackedSession[*ChaseGameState, any, string]) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getCurrentTime (GetCurrentTime)
	getCurrentTime_timestamp := time.Now().Unix()
	// Node: getGameStart (GetField)
	getGameStart_value := state.GameStartTime()
	// Node: getGameDuration (GetField)
	getGameDuration_value := state.GameDurationHours()
	// Node: calcDurationSec (Multiply)
	calcDurationSec_result := getGameDuration_value * 3600
	// Node: calcEndTime (Add)
	calcEndTime_result := getGameStart_value + calcDurationSec_result
	// Node: checkTimeUp (Compare)
	checkTimeUp_result := getCurrentTime_timestamp >= calcEndTime_result
	// Node: ifTimeUp (If)
	if checkTimeUp_result {
		// Node: setGameOver (SetField)
		state.SetIsGameOver(true)
		// Node: setWinner (SetField)
		state.SetWinningTeam("runner")
		// Node: emitGameOver (EmitToAll)
		session.Emit("GameOver", nil)
	} else {
	}
	return nil
}

// OnTrySpawnBox handles the TrySpawnBox event
func OnTrySpawnBox(session *statesync.TrackedSession[*ChaseGameState, any, string]) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getCurrentTime (GetCurrentTime)
	getCurrentTime_timestamp := time.Now().Unix()
	// Node: getNextSpawnTime (GetField)
	getNextSpawnTime_value := state.NextBoxSpawnTime()
	// Node: getBoxesSpawned (GetField)
	getBoxesSpawned_value := state.BoxesSpawned()
	// Node: getMaxBoxes (GetField)
	getMaxBoxes_value := state.MaxSurpriseBoxes()
	// Node: checkTimeToSpawn (Compare)
	checkTimeToSpawn_result := getCurrentTime_timestamp >= getNextSpawnTime_value
	// Node: checkCanSpawnMore (Compare)
	checkCanSpawnMore_result := getBoxesSpawned_value < getMaxBoxes_value
	// Node: canSpawn (And)
	canSpawn_result := checkTimeToSpawn_result && checkCanSpawnMore_result
	// Node: ifCanSpawn (If)
	if canSpawn_result {
		// Node: randomLat (RandomFloat)
		randomLat_result := rand.Float64()*(47.55-47.45) + 47.45
		// Node: randomLng (RandomFloat)
		randomLng_result := rand.Float64()*(19.1-19) + 19
		// Node: randomPoints (RandomInt)
		randomPoints_result := rand.Intn(100-10+1) + 10
		// Node: randomDelay (RandomInt)
		randomDelay_result := rand.Intn(300-60+1) + 60
		// Node: incrementBoxCount (Add)
		incrementBoxCount_result := getBoxesSpawned_value + 1
		// Node: createBox (CreateStruct)
		createBox_result := SurpriseBox{
			PointValue:  randomPoints_result,
			ID:          incrementBoxCount_result,
			Lat:         randomLat_result,
			Lng:         randomLng_result,
			SpawnTime:   getCurrentTime_timestamp,
			IsCollected: false,
			CollectedBy: "",
		}
		// Node: appendBox (ArrayAppend)
		state.AppendSurpriseBoxes(createBox_result)
		// Node: setBoxesSpawned (SetField)
		state.SetBoxesSpawned(incrementBoxCount_result)
		// Node: calcNextSpawn (Add)
		calcNextSpawn_result := getCurrentTime_timestamp + randomDelay_result
		// Node: setNextSpawn (SetField)
		state.SetNextBoxSpawnTime(calcNextSpawn_result)
		// Node: emitBoxSpawned (EmitToAll)
		session.Emit("SurpriseBoxSpawned", nil)
	} else {
	}
	return nil
}

// OnPlayerUpdatePosition handles the PlayerUpdatePosition event
func OnPlayerUpdatePosition(session *statesync.TrackedSession[*ChaseGameState, any, string], playerID string, lat float64, lng float64) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getGameZone (GetField)
	getGameZone_value := state.GameZone()
	// Node: checkInZone (PointInPolygon)
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
	if checkInZone_isInside {
		// Node: updatePlayerPos (UpdateWhere)
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
		session.EmitTo(playerID, "PositionAccepted", nil)
	} else {
		// Node: emitOutOfZone (EmitToPlayer)
		session.EmitTo(playerID, "OutOfGameZone", nil)
	}
	return nil
}

// OnTryCollectBox handles the TryCollectBox event
func OnTryCollectBox(session *statesync.TrackedSession[*ChaseGameState, any, string], playerID string, boxID string) error {
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
	// Node: getPlayerLat (GetStructField)
	getPlayerLat_value := getPlayer_player.Lat
	// Node: getPlayerLng (GetStructField)
	getPlayerLng_value := getPlayer_player.Lng
	// Node: iterateBoxes (ForEachWhere)
	for iterateBoxes_index := 0; iterateBoxes_index < state.SurpriseBoxesLen(); iterateBoxes_index++ {
		iterateBoxes_item := state.SurpriseBoxesAt(iterateBoxes_index)
		if iterateBoxes_item.ID == boxID {
			// Node: getBoxLat (GetStructField)
			getBoxLat_value := iterateBoxes_item.Lat
			// Node: getBoxLng (GetStructField)
			getBoxLng_value := iterateBoxes_item.Lng
			// Node: checkNotCollected (GetStructField)
			checkNotCollected_value := iterateBoxes_item.IsCollected
			// Node: isNotCollected (Not)
			isNotCollected_result := !checkNotCollected_value
			// Node: checkInRange (PointInCircle)
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
			canCollect_result := isNotCollected_result && checkInRange_isInside
			// Node: ifCanCollect (If)
			if canCollect_result {
				// Node: collectBox (UpdateStruct)
				iterateBoxes_item.IsCollected = true
				iterateBoxes_item.CollectedBy = playerID
				// Node: emitCollected (EmitToPlayer)
				session.EmitTo(playerID, "BoxCollected", nil)
			}
			state.UpdateSurpriseBoxesAt(iterateBoxes_index, iterateBoxes_item)
		}
	}
	return nil
}

// OnCheckCapture handles the CheckCapture event
func OnCheckCapture(session *statesync.TrackedSession[*ChaseGameState, any, string], chaserID string) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getChaser (GetPlayer)
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
	getChaserLat_value := getChaser_player.Lat
	// Node: getChaserLng (GetStructField)
	getChaserLng_value := getChaser_player.Lng
	// Node: getCaptureRadius (GetField)
	getCaptureRadius_value := state.CaptureRadius()
	// Node: iterateRunners (ForEachWhere)
	for iterateRunners_index := 0; iterateRunners_index < state.PlayersLen(); iterateRunners_index++ {
		iterateRunners_item := state.PlayersAt(iterateRunners_index)
		if iterateRunners_item.Team == "runner" {
			// Node: getRunnerCaptured (GetStructField)
			getRunnerCaptured_value := iterateRunners_item.IsCaptured
			// Node: isNotCaptured (Not)
			isNotCaptured_result := !getRunnerCaptured_value
			// Node: ifNotCaptured (If)
			if isNotCaptured_result {
				// Node: getRunnerLat (GetStructField)
				getRunnerLat_value := iterateRunners_item.Lat
				// Node: getRunnerLng (GetStructField)
				getRunnerLng_value := iterateRunners_item.Lng
				// Node: checkInCaptureRange (PointInCircle)
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
				if checkInCaptureRange_isInside {
					// Node: captureRunner (UpdateStruct)
					iterateRunners_item.IsCaptured = true
					// Node: emitCaptured (EmitToAll)
					session.Emit("RunnerCaptured", nil)
				}
			}
			state.UpdatePlayersAt(iterateRunners_index, iterateRunners_item)
		}
	}
	return nil
}

// OnUpdatePublicPositions handles the UpdatePublicPositions event
func OnUpdatePublicPositions(session *statesync.TrackedSession[*ChaseGameState, any, string]) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getCurrentTime (GetCurrentTime)
	getCurrentTime_timestamp := time.Now().Unix()
	// Node: iteratePlayers (ForEachWhere)
	for iteratePlayers_index := 0; iteratePlayers_index < state.PlayersLen(); iteratePlayers_index++ {
		iteratePlayers_item := state.PlayersAt(iteratePlayers_index)
		if iteratePlayers_item.ID != "" {
			// Node: getLastUpdate (GetStructField)
			getLastUpdate_value := iteratePlayers_item.LastPublicUpdate
			// Node: calcTimeSince (Subtract)
			calcTimeSince_result := getCurrentTime_timestamp - getLastUpdate_value
			// Node: checkNeedsUpdate (Compare)
			checkNeedsUpdate_result := calcTimeSince_result >= 60
			// Node: ifNeedsUpdate (If)
			if checkNeedsUpdate_result {
				// Node: getPrivateLat (GetStructField)
				getPrivateLat_value := iteratePlayers_item.Lat
				// Node: getPrivateLng (GetStructField)
				getPrivateLng_value := iteratePlayers_item.Lng
				// Node: updatePublicPos (UpdateStruct)
				iteratePlayers_item.PublicLat = getPrivateLat_value
				iteratePlayers_item.PublicLng = getPrivateLng_value
				iteratePlayers_item.LastPublicUpdate = getCurrentTime_timestamp
			}
			state.UpdatePlayersAt(iteratePlayers_index, iteratePlayers_item)
		}
	}
	return nil
}
