package fsm

import (
	"Sanntid/elevio"
)

type ElevatorState int

const (
	Idle ElevatorState = iota
	MovingBetweenFloors
	MovingPassingFloor
	DoorOpen
)

type ElevatorEvents struct {
	NewOrder       elevio.ButtonEvent
	ArrivedAtFloor int
	DoorTimeout    bool
}

type ElevatorDescision struct {
	NextState      ElevatorState
	ElevatorOutput ElevatorOutput
}

type ElevatorOutput struct {
	MotorDirection elevio.MotorDirection
	Door           bool
	LocalOrders    [NFloors][NButtons]bool
}

type ElevatorInput struct {
	LocalRequests [NFloors][NButtons]bool
	PrevFloor     int
}
type Elevator struct {
	State  ElevatorState
	Input  ElevatorInput
	Output ElevatorOutput
}

func RequestsAbove(elevator ElevatorInput) bool {
	for i := elevator.PrevFloor + 1; i < 4; i++ {
		if elevator.LocalRequests[i][0] || elevator.LocalRequests[i][1] || elevator.LocalRequests[i][2] {
			return true
		}
	}
	return false
}

func RequestsBelow(elevator ElevatorInput) bool {
	for i := 0; i < elevator.PrevFloor; i++ {
		if elevator.LocalRequests[i][0] || elevator.LocalRequests[i][1] || elevator.LocalRequests[i][2] {
			return true
		}
	}
	return false
}

func QueueEmpty(queue [4][3]bool) bool {
	for i := 0; i < NButtons; i++ {
		for j := 0; j < NFloors; j++ {
			if queue[j][i] {
				return false
			}
		}
	}
	return true
}

func HandleFloorReached(event int, storedInput ElevatorInput, storedOutput ElevatorOutput) ElevatorDescision {
	var nextState ElevatorState
	var Output ElevatorOutput
	storedInput.PrevFloor = event

	if QueueEmpty(storedInput.LocalRequests) {
		print("Queue empty")
		nextState = DoorOpen
		Output.MotorDirection = elevio.MD_Stop
		Output.LocalOrders = storedInput.LocalRequests
		Output.Door = true
		return ElevatorDescision{nextState, Output}
	}

	caseDown := storedOutput.MotorDirection == elevio.MD_Down && (storedOutput.LocalOrders[event][1] || storedOutput.LocalOrders[event][2] || !RequestsBelow(storedInput))
	caseUp := storedOutput.MotorDirection == elevio.MD_Up && (storedOutput.LocalOrders[event][0] || storedOutput.LocalOrders[event][2] || !RequestsAbove(storedInput))

	if caseDown {
		print("caseDown")
		nextState = DoorOpen
		Output.MotorDirection = elevio.MD_Stop
		Output.Door = true
		Output.LocalOrders = storedInput.LocalRequests

		if !RequestsBelow(storedInput) {
			Output.LocalOrders[event][0] = false
		}
		Output.LocalOrders[event][1] = false
		Output.LocalOrders[event][2] = false
		storedInput.LocalRequests = storedOutput.LocalOrders
		return ElevatorDescision{nextState, Output}
	}
	if caseUp {
		print("caseUp")
		nextState = DoorOpen
		Output.MotorDirection = elevio.MD_Stop
		Output.Door = true
		Output.LocalOrders = storedInput.LocalRequests
		if !RequestsAbove(storedInput) {
			Output.LocalOrders[event][1] = false
		}
		if storedInput.LocalRequests[event][2] {
			Output.LocalOrders[event][0] = false
			Output.LocalOrders[event][2] = false
			storedInput.LocalRequests = storedOutput.LocalOrders
			nextState = DoorOpen
		}
		return ElevatorDescision{nextState, Output}
	}
	print("ingen case")
	storedOutput.LocalOrders = storedInput.LocalRequests
	storedOutput.LocalOrders[event][2] = false

	return ElevatorDescision{MovingPassingFloor, storedOutput}
}

func HandleDoorTimeout(storedInput ElevatorInput, storedOutput ElevatorOutput) ElevatorDescision {
	var nextState ElevatorState
	var Output ElevatorOutput
	if QueueEmpty(storedInput.LocalRequests) {
		Output.Door = false
		nextState = Idle

	} else {
		switch storedOutput.MotorDirection {
		case elevio.MD_Stop:
			if RequestsAbove(storedInput) {
				Output.MotorDirection = elevio.MD_Up
				nextState = MovingBetweenFloors
			} else if RequestsBelow(storedInput) {
				Output.MotorDirection = elevio.MD_Down
				nextState = MovingBetweenFloors
			} else {
				Output.MotorDirection = elevio.MD_Stop
				nextState = MovingBetweenFloors
			}
		case elevio.MD_Up:
			if RequestsAbove(storedInput) {
				Output.MotorDirection = elevio.MD_Up
				nextState = MovingBetweenFloors
			} else if RequestsBelow(storedInput) && (storedInput.LocalRequests[storedInput.PrevFloor][0]) {
				nextState = DoorOpen
				Output.Door = true
				Output.MotorDirection = elevio.MD_Stop
			} else if RequestsBelow(storedInput) {
				Output.MotorDirection = elevio.MD_Down
				nextState = MovingBetweenFloors
			} else {
				Output.MotorDirection = elevio.MD_Stop
				nextState = MovingBetweenFloors
			}

		case elevio.MD_Down:
			if RequestsBelow(storedInput) {
				Output.MotorDirection = elevio.MD_Down
				nextState = MovingBetweenFloors
			} else if RequestsAbove(storedInput) && (storedInput.LocalRequests[storedInput.PrevFloor][1]) {
				nextState = DoorOpen
				Output.Door = true
				Output.MotorDirection = elevio.MD_Stop
			} else if RequestsAbove(storedInput) {
				Output.MotorDirection = elevio.MD_Up
				nextState = MovingBetweenFloors
			} else {
				Output.MotorDirection = elevio.MD_Stop
				nextState = MovingBetweenFloors
			}
		}

	}
	Output.LocalOrders = storedOutput.LocalOrders
	return ElevatorDescision{nextState, Output}
}
