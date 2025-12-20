package drone

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

// OnAddDrone handles the AddDrone event
func OnAddDrone(session *statesync.TrackedSession[*DroneState, any, string], playerID string, droneID string) error {
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
	// Node: createDrone (CreateStruct)
	createDrone_result := Drone{
		TargetLng: 0,
		HasTarget: false,
		Speed:     4.166666666666667,
		ID:        droneID,
		OwnerID:   playerID,
		Lat:       getPlayerLat_value,
		Lng:       getPlayerLng_value,
		TargetLat: 0,
	}
	// Node: appendDrone (ArrayAppend)
	state.AppendDrones(createDrone_result)
	// Node: emitDroneAdded (EmitToAll)
	session.Emit("DroneAdded", nil)
	return nil
}

// OnSetDroneTarget handles the SetDroneTarget event
func OnSetDroneTarget(session *statesync.TrackedSession[*DroneState, any, string], droneID string, targetLat float64, targetLng float64) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: updateDrone (UpdateWhere)
	updateDrone_count := 0
	for _i := 0; _i < state.DronesLen(); _i++ {
		_item := state.DronesAt(_i)
		if _item.ID == droneID {
			_item.TargetLat = targetLat
			_item.TargetLng = targetLng
			_item.HasTarget = true
			state.UpdateDronesAt(_i, _item)
			updateDrone_count++
		}
	}
	// Node: emitTargetSet (EmitToAll)
	session.Emit("DroneTargetSet", nil)
	return nil
}

// OnTick handles the Tick event
func OnTick(session *statesync.TrackedSession[*DroneState, any, string]) error {
	state := session.State().Get()
	_ = state // may be unused

	// Node: getTick (GetField)
	getTick_value := state.TickCount()
	// Node: addTick (Add)
	addTick_result := getTick_value + 1
	// Node: setTick (SetField)
	state.SetTickCount(addTick_result)
	// Node: iterateDrones (ForEachWhere)
	for iterateDrones_index := 0; iterateDrones_index < state.DronesLen(); iterateDrones_index++ {
		iterateDrones_item := state.DronesAt(iterateDrones_index)
		if iterateDrones_item.HasTarget == true {
			// Node: getDroneLat (GetStructField)
			getDroneLat_value := iterateDrones_item.Lat
			// Node: getDroneLng (GetStructField)
			getDroneLng_value := iterateDrones_item.Lng
			// Node: getTargetLat (GetStructField)
			getTargetLat_value := iterateDrones_item.TargetLat
			// Node: getTargetLng (GetStructField)
			getTargetLng_value := iterateDrones_item.TargetLng
			// Node: calcDistance (GpsDistance)
			// Haversine distance calculation
			_calcDistance_lat1Rad := getDroneLat_value * math.Pi / 180.0
			_calcDistance_lat2Rad := getTargetLat_value * math.Pi / 180.0
			_calcDistance_deltaLat := (getTargetLat_value - getDroneLat_value) * math.Pi / 180.0
			_calcDistance_deltaLng := (getTargetLng_value - getDroneLng_value) * math.Pi / 180.0
			_calcDistance_a := math.Sin(_calcDistance_deltaLat/2)*math.Sin(_calcDistance_deltaLat/2) + math.Cos(_calcDistance_lat1Rad)*math.Cos(_calcDistance_lat2Rad)*math.Sin(_calcDistance_deltaLng/2)*math.Sin(_calcDistance_deltaLng/2)
			_calcDistance_c := 2 * math.Atan2(math.Sqrt(_calcDistance_a), math.Sqrt(1-_calcDistance_a))
			calcDistance_distance := 6371000.0 * _calcDistance_c
			// Node: checkNotArrived (Compare)
			checkNotArrived_result := calcDistance_distance > 1
			// Node: ifNotArrived (If)
			if checkNotArrived_result {
				// Node: moveToward (GpsMoveToward)
				// Calculate bearing and move toward target
				_moveToward_fromLatRad := getDroneLat_value * math.Pi / 180.0
				_moveToward_fromLngRad := getDroneLng_value * math.Pi / 180.0
				_moveToward_toLatRad := getTargetLat_value * math.Pi / 180.0
				_moveToward_toLngRad := getTargetLng_value * math.Pi / 180.0
				_moveToward_dLng := _moveToward_toLngRad - _moveToward_fromLngRad
				_moveToward_x := math.Cos(_moveToward_toLatRad) * math.Sin(_moveToward_dLng)
				_moveToward_y := math.Cos(_moveToward_fromLatRad)*math.Sin(_moveToward_toLatRad) - math.Sin(_moveToward_fromLatRad)*math.Cos(_moveToward_toLatRad)*math.Cos(_moveToward_dLng)
				_moveToward_bearing := math.Atan2(_moveToward_x, _moveToward_y)
				_moveToward_angularDist := 12.5 / 6371000.0
				_moveToward_newLatRad := math.Asin(math.Sin(_moveToward_fromLatRad)*math.Cos(_moveToward_angularDist) + math.Cos(_moveToward_fromLatRad)*math.Sin(_moveToward_angularDist)*math.Cos(_moveToward_bearing))
				_moveToward_newLngRad := _moveToward_fromLngRad + math.Atan2(math.Sin(_moveToward_bearing)*math.Sin(_moveToward_angularDist)*math.Cos(_moveToward_fromLatRad), math.Cos(_moveToward_angularDist)-math.Sin(_moveToward_fromLatRad)*math.Sin(_moveToward_newLatRad))
				moveToward_newLat := _moveToward_newLatRad * 180.0 / math.Pi
				moveToward_newLng := _moveToward_newLngRad * 180.0 / math.Pi
				// Node: updateDronePos (UpdateStruct)
				iterateDrones_item.Lat = moveToward_newLat
				iterateDrones_item.Lng = moveToward_newLng
			} else {
				// Node: clearTarget (UpdateStruct)
				iterateDrones_item.HasTarget = false
			}
			state.UpdateDronesAt(iterateDrones_index, iterateDrones_item)
		}
	}
	// Node: emitDronesMoved (EmitToAll)
	session.Emit("DronesMoved", nil)
	return nil
}
