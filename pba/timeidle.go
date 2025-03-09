package pba

import (
	"Sanntid/fsm"
	"math"
)

/*
type Direction int
type ElevatorBehaviour int
type Button int

const (
	Up Direction = iota + 1
	Down
	Stop
)

const (
	Idle ElevatorBehaviour = iota
	Moving
	DoorOpen
)

const (
	N_FLOORS       = 4
	N_BUTTONS      = 3
	TRAVEL_TIME    = 2
	DOOR_OPEN_TIME = 3
)

type Elevator struct {
	floor     int
	dirn      Direction
	requests  [N_FLOORS][N_BUTTONS]bool
	behaviour ElevatorBehaviour
}

func requestsChooseDirection(e Elevator) Direction {
	for i := e.floor + 1; i < N_FLOORS; i++ {
		for j := 0; j < N_BUTTONS; j++ {
			if e.requests[i][j] {
				return Up
			}
		}
	}

	for i := e.floor - 1; i >= 0; i-- {
		for j := 0; j < N_BUTTONS; j++ {
			if e.requests[i][j] {
				return Down
			}
		}
	}

	return Stop

}

func requestsShouldStop(e Elevator) bool {
	for j := 0; j < N_BUTTONS; j++ {
		if e.requests[e.floor][j] {
			return true
		}
	}
	return false
}

func requestsClearAtCurrentFloor(e Elevator, onClearedRequest func(Buttbehaviour ElevatorBehaviouron, int)) Elevator {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.requests[e.floor][btn] {
			e.requests[e.floor][btn] = false
			if onClearedRequest != nil {
				onClearedRequest(Button(btn), e.floor)
			}
		}
	}
	return e
}

func requestsClearAtCurrentFloor(e Elevator) {
	caseDown := e.dirn == 2 && (e.requests[e.floor][1] || e.requests[e.floor][2] || !requestsBelow(e))
	caseUp := e.dirn == 0 && (e.requests[e.floor][0] || e.requests[e.floor][2] || !requestsAbove(e))

	if caseDown || caseUp {
		fmt.Println("Stopping at floor", e.floor)
		e.requests[e.floor][2] = false
		if caseDown {
			if !requestsBelow(e) {
				e.requests[e.floor][0] = false
			}
			e.requests[e.floor][1] = false
		}
		if caseUp {
			if !requestsAbove(e) {
				e.requests[e.floor][1] = false
			}
			e.requests[e.floor][0] = false
		}
	}
}

func timeToIdle(e Elevator) int {
	timer := 0

	switch e.behaviour {
	case Idle:
		e.dirn = requestsChooseDirection(e)
		if e.dirn == Stop {
			return timer
		}
	case Moving:
		timer += TRAVEL_TIME
		e.floor += int(e.dirn)
	case DoorOpen:
		timer += DOOR_OPEN_TIME
	}

	for {
		if requestsShouldStop(e) {
			e = requestsClearAtCurrentFloor(e, nil) //,nil)
			timer += DOOR_OPEN_TIME
			e.dirn = requestsChooseDirection(e)
			if e.dirn == Stop {
				return timer
			}
		}
		e.floor += int(e.dirn)
		timer += TRAVEL_TIME
	}
}

		floor:     0,
		dirn:      Stop,
		behaviour: Idle,
	}

	queue := [N_FLOORS][N_BUTTONS]bool{
		{false, false, false},
		{true, true, true},
		{false, false, false},
		{false, false, true},
	}

	e.requests = queue
	duration := timeToIdle(e)
	fmt.Printf("Time to idle: %d\n", duration)
}
*/

// min function to find the minimum value in an array
func min(arr []int) int {
	minVal := math.MaxInt64
	for _, value := range arr {
		if value < minVal {
			minVal = value
		}
	}
	return minVal
}

// indexOf function to find the index of the minimum value
func indexOf(arr []int, value int) int {
	for i, v := range arr {
		if v == value {
			return i
		}
	}
	return -1 // Return -1 if the value is not found
}

func AssignRequest(order fsm.Order, status [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool) ([fsm.NFloors][fsm.NButtons][fsm.MElevators]bool, int) {
	numFloorsToIdle := 0
	numDoorOpensToIdle := 0
	prevOrderFloor := 0
	completeTimes := make([]int, fsm.MElevators) // Declare completeTimes as an array
	

	for k := 0; k < fsm.MElevators; k++ {
		numFloorsToIdle = 0
		numDoorOpensToIdle = 0
		prevOrderFloor = 0
		for i := 0; i < fsm.NFloors; i++ {
			for j := 0; j < fsm.NButtons; j++ {
				if status[i][j][k] {
					numDoorOpensToIdle++
					numFloorsToIdle += int(math.Abs(float64(i - prevOrderFloor)))
					prevOrderFloor = i
				}
			}
		}
		completeTimes[k] = 3*numDoorOpensToIdle + numFloorsToIdle

	}

	minTime := min(completeTimes)
	responsibleElevator := indexOf(completeTimes, minTime)
	status[order.ButtonEvent.Floor][order.ButtonEvent.Button][responsibleElevator] = true

	return status, responsibleElevator
}