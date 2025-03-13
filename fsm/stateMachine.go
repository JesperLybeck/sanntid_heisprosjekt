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
	ButtonLights   [4][3]bool
}

type ElevatorInput struct {
	LocalRequests [NFloors][NButtons]bool
	PrevFloor     int
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
	var nextOutput ElevatorOutput
	storedInput.PrevFloor = event

	if QueueEmpty(storedInput.LocalRequests) {
		print("Queue empty")
		nextState = DoorOpen
		nextOutput.MotorDirection = elevio.MD_Stop
		nextOutput.ButtonLights = storedInput.LocalRequests
		nextOutput.Door = true
		return ElevatorDescision{nextState, nextOutput}
	}

	caseDown := storedOutput.MotorDirection == elevio.MD_Down && (storedOutput.ButtonLights[event][1] || storedOutput.ButtonLights[event][2] || !RequestsBelow(storedInput))
	caseUp := storedOutput.MotorDirection == elevio.MD_Up && (storedOutput.ButtonLights[event][0] || storedOutput.ButtonLights[event][2] || !RequestsAbove(storedInput))

	if caseDown {
		print("caseDown")
		nextState = DoorOpen
		nextOutput.MotorDirection = elevio.MD_Stop
		nextOutput.Door = true
		nextOutput.ButtonLights = storedInput.LocalRequests

		if !RequestsBelow(storedInput) {
			nextOutput.ButtonLights[event][0] = false
		}
		nextOutput.ButtonLights[event][1] = false
		nextOutput.ButtonLights[event][2] = false
		storedInput.LocalRequests = storedOutput.ButtonLights
		return ElevatorDescision{nextState, nextOutput}
	}
	if caseUp {
		print("caseUp")
		nextState = DoorOpen
		nextOutput.MotorDirection = elevio.MD_Stop
		nextOutput.Door = true
		nextOutput.ButtonLights = storedInput.LocalRequests
		if !RequestsAbove(storedInput) {
			nextOutput.ButtonLights[event][1] = false
		}
		if storedInput.LocalRequests[event][2] {
			nextOutput.ButtonLights[event][0] = false
			nextOutput.ButtonLights[event][2] = false
			storedInput.LocalRequests = storedOutput.ButtonLights
			nextState = DoorOpen
		}
		return ElevatorDescision{nextState, nextOutput}
	}
	print("ingen case")
	storedOutput.ButtonLights = storedInput.LocalRequests
	storedOutput.ButtonLights[event][2] = false

	return ElevatorDescision{MovingPassingFloor, storedOutput}
}

func HandleDoorTimeout(storedInput ElevatorInput, storedOutput ElevatorOutput) ElevatorDescision {
	var nextState ElevatorState
	var nextOutput ElevatorOutput
	if QueueEmpty(storedInput.LocalRequests) {
		nextOutput.Door = false
		nextState = Idle

	} else {
		switch storedOutput.MotorDirection {
		case elevio.MD_Stop:
			if RequestsAbove(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Up
				nextState = MovingBetweenFloors
			} else if RequestsBelow(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Down
				nextState = MovingBetweenFloors
			} else {
				nextOutput.MotorDirection = elevio.MD_Stop
				nextState = MovingBetweenFloors
			}
		case elevio.MD_Up:
			if RequestsAbove(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Up
				nextState = MovingBetweenFloors
			} else if RequestsBelow(storedInput) && (storedInput.LocalRequests[storedInput.PrevFloor][0]) {
				nextState = DoorOpen
				nextOutput.Door = true
				nextOutput.MotorDirection = elevio.MD_Stop
			} else if RequestsBelow(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Down
				nextState = MovingBetweenFloors
			} else {
				nextOutput.MotorDirection = elevio.MD_Stop
				nextState = MovingBetweenFloors
			}

		case elevio.MD_Down:
			if RequestsBelow(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Down
				nextState = MovingBetweenFloors
			} else if RequestsAbove(storedInput) && (storedInput.LocalRequests[storedInput.PrevFloor][1]) {
				nextState = DoorOpen
				nextOutput.Door = true
				nextOutput.MotorDirection = elevio.MD_Stop
			} else if RequestsAbove(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Up
				nextState = MovingBetweenFloors
			} else {
				nextOutput.MotorDirection = elevio.MD_Stop
				nextState = MovingBetweenFloors
			}
		}

	}
	nextOutput.ButtonLights = storedOutput.ButtonLights
	return ElevatorDescision{nextState, nextOutput}
}
