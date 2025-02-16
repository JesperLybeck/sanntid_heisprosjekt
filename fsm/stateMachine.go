package fsm

import (
	"Sanntid/elevio"

	"fmt"
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
	PressedButtons [4][3]bool
	PrevFloor      int
}

func RequestsAbove(elevator ElevatorInput) bool {
	for i := elevator.PrevFloor + 1; i < 4; i++ {
		if elevator.PressedButtons[i][0] || elevator.PressedButtons[i][1] || elevator.PressedButtons[i][2] {
			return true
		}
	}
	return false
}

func RequestsBelow(elevator ElevatorInput) bool {
	for i := 0; i < elevator.PrevFloor; i++ {
		if elevator.PressedButtons[i][0] || elevator.PressedButtons[i][1] || elevator.PressedButtons[i][2] {
			return true
		}
	}
	return false
}

func HandleFloorReached(event int, storedInput ElevatorInput, storedOutput ElevatorOutput) ElevatorDescision {
	println("motor direction at floor reached", storedOutput.MotorDirection)
	var nextState ElevatorState
	var nextOutput ElevatorOutput
	var QueueNotEmpty bool = false
	var numLights int = 0
	fmt.Println("buttonlights", storedOutput.ButtonLights)
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			if storedOutput.ButtonLights[j][i] {
				QueueNotEmpty = true
				numLights++
				break
			}
		}
	}
	if !QueueNotEmpty {
		nextState = DoorOpen
		nextOutput.MotorDirection = elevio.MD_Stop
		nextOutput.ButtonLights = storedInput.PressedButtons
		nextOutput.Door = true
		println("Queue is empty, stopping at floor", event)
		return ElevatorDescision{nextState, nextOutput}
	}

	//if (storedInput.PressedButtons[event][0] && storedOutput.MotorDirection == elevio.MD_Up) || (storedInput.PressedButtons[event][1] && storedOutput.MotorDirection == elevio.MD_Down) || (storedInput.PressedButtons[event][2]) {
	isCabCall := storedOutput.ButtonLights[event][2]
	isOrderInDirectionUp := storedOutput.ButtonLights[event][0] && storedOutput.MotorDirection == elevio.MD_Up
	isOrderInDirectionDown := storedOutput.ButtonLights[event][1] && storedOutput.MotorDirection == elevio.MD_Down
	isOrderFirstFloor := (storedOutput.ButtonLights[0][0] || storedOutput.ButtonLights[0][1] || storedOutput.ButtonLights[0][2]) && event == 0
	isOrderTopFloor := (storedOutput.ButtonLights[NFloors-1][0] || storedOutput.ButtonLights[NFloors-1][1] || storedOutput.ButtonLights[NFloors-1][2]) && event == NFloors
	isOnlyOrder := numLights == 1 && (storedOutput.ButtonLights[event][0] || storedOutput.ButtonLights[event][1] || storedOutput.ButtonLights[event][2])

	println("isCabCall", isCabCall, "isOrderInDirectionUp", isOrderInDirectionUp, "isOrderInDirectionDown", isOrderInDirectionDown, "isOrderFirstFloor", isOrderFirstFloor, "isOrderTopFloor", isOrderTopFloor)

	if isCabCall || isOrderInDirectionUp || isOrderInDirectionDown || isOrderFirstFloor || isOrderTopFloor || isOnlyOrder {
		fmt.Println("stopping at floor", event)
		nextState = DoorOpen
		nextOutput.MotorDirection = elevio.MD_Stop
		nextOutput.Door = true
		nextOutput.ButtonLights = storedInput.PressedButtons
		nextOutput.ButtonLights[event][0] = false
		nextOutput.ButtonLights[event][1] = false
		nextOutput.ButtonLights[event][2] = false
		storedInput.PressedButtons = storedOutput.ButtonLights
		return ElevatorDescision{nextState, nextOutput}
	}
	fmt.Println("decided direction", storedOutput.MotorDirection)
	return ElevatorDescision{MovingPassingFloor, storedOutput}
}

func HandleNewOrder(state ElevatorState, event elevio.ButtonEvent, storedInput ElevatorInput, storedOutput ElevatorOutput) ElevatorDescision {

	var nextState ElevatorState
	var nextOutput ElevatorOutput
	println("new order, state:", state)
	if state == Idle {
		fmt.Println(storedInput.PrevFloor, event.Floor)
		if storedInput.PrevFloor < event.Floor {
			nextOutput.MotorDirection = elevio.MD_Up
			nextOutput.ButtonLights = storedInput.PressedButtons
			nextState = MovingBetweenFloors

		} else if storedInput.PrevFloor > event.Floor {
			nextOutput.MotorDirection = elevio.MD_Down
			nextOutput.ButtonLights = storedInput.PressedButtons
			nextState = MovingBetweenFloors

		} else {
			nextOutput.MotorDirection = elevio.MD_Stop
			nextState = DoorOpen
			nextOutput.Door = true
			nextOutput.ButtonLights = storedInput.PressedButtons
			nextOutput.ButtonLights[event.Floor][0] = false
			nextOutput.ButtonLights[event.Floor][1] = false
			nextOutput.ButtonLights[event.Floor][2] = false
			println("evetnfloor", event.Floor)
			storedInput.PressedButtons = nextOutput.ButtonLights
			//println(3)

		}
		return ElevatorDescision{nextState, nextOutput}
	} else if state == DoorOpen {
		nextOutput.ButtonLights = storedInput.PressedButtons
		nextOutput.MotorDirection = elevio.MD_Stop
		fmt.Println("New order at door open", storedInput.PrevFloor, event.Floor)

	} else {
		fmt.Println("BITCH", storedInput.PrevFloor, event.Floor)
		fmt.Println("BITCH", storedInput.PressedButtons)
		nextOutput.ButtonLights = storedInput.PressedButtons
		nextOutput.MotorDirection = storedOutput.MotorDirection
		fmt.Println(storedOutput.MotorDirection)

	}
	return ElevatorDescision{state, nextOutput}

}
func HandleDoorTimeout(storedInput ElevatorInput, storedOutput ElevatorOutput) ElevatorDescision {
	var nextState ElevatorState
	var nextOutput ElevatorOutput
	unservedOrders := false
	for i := 0; i < 4; i++ {
		if storedInput.PressedButtons[i][0] || storedInput.PressedButtons[i][1] || storedInput.PressedButtons[i][2] {
			unservedOrders = true
			break
		}
	}
	if !unservedOrders {
		nextOutput.Door = false
		nextState = Idle

	} else {

		switch storedOutput.MotorDirection {
		case elevio.MD_Stop:
			if RequestsAbove(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Up
			} else if RequestsBelow(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Down
			} else {
				nextOutput.MotorDirection = elevio.MD_Stop
			}
		case elevio.MD_Up:
			if RequestsAbove(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Up
			} else if RequestsBelow(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Down
			} else {
				nextOutput.MotorDirection = elevio.MD_Stop
			}

		case elevio.MD_Down:
			if RequestsBelow(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Down
			} else if RequestsAbove(storedInput) {
				nextOutput.MotorDirection = elevio.MD_Up
			} else {
				nextOutput.MotorDirection = elevio.MD_Stop
			}
		}
	}
	nextOutput.ButtonLights = storedOutput.ButtonLights
	fmt.Println("motordirection decided be door timeout", nextOutput.MotorDirection)
	return ElevatorDescision{nextState, nextOutput}
}
