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

type DirectionStatePair struct {
	Direction elevio.MotorDirection
	State     ElevatorState
}

func OrdersAbove(E Elevator) bool {
	print("checking orders above")
	for i := E.Input.PrevFloor + 1; i < NFloors; i++ {
		if E.Output.LocalOrders[i][0] || E.Output.LocalOrders[i][1] || E.Output.LocalOrders[i][2] {
			println("orders above")
			return true
		}
	}
	return false
}

func OrdersBelow(E Elevator) bool {
	for i := 0; i < E.Input.PrevFloor; i++ {
		if E.Output.LocalOrders[i][0] || E.Output.LocalOrders[i][1] || E.Output.LocalOrders[i][2] {
			return true
		}
	}
	return false
}
func OrdersHere(E Elevator) bool {
	return E.Output.LocalOrders[E.Input.PrevFloor][0] || E.Output.LocalOrders[E.Input.PrevFloor][1] || E.Output.LocalOrders[E.Input.PrevFloor][2]
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

func shouldStop(E Elevator) bool {
	println("checking if should stop")
	switch E.Output.MotorDirection {
	case elevio.MD_Down:
		return (E.Output.LocalOrders[E.Input.PrevFloor][elevio.BT_HallDown] ||
			E.Output.LocalOrders[E.Input.PrevFloor][elevio.BT_Cab] ||
			!OrdersBelow(E))
	case elevio.MD_Up:
		println("elevator going up")
		return (E.Output.LocalOrders[E.Input.PrevFloor][elevio.BT_HallUp] ||
			E.Output.LocalOrders[E.Input.PrevFloor][elevio.BT_Cab] ||
			!OrdersAbove(E))
	case elevio.MD_Stop:
		return true

	}
	return true

}

// checking if the order is to be cleared immediately.
func shouldClearImmediately(E Elevator, btnEvent elevio.ButtonEvent) bool {
	fmt.Println(E.Input.PrevFloor, " ", btnEvent.Floor, " ", E.Output.MotorDirection, " ", btnEvent.Button)
	return ((E.Input.PrevFloor == btnEvent.Floor) && ((E.Output.MotorDirection == elevio.MD_Up && btnEvent.Button == elevio.BT_HallUp) ||
		(E.Output.MotorDirection == elevio.MD_Down && btnEvent.Button == elevio.BT_HallDown) ||
		(E.Output.MotorDirection == elevio.MD_Stop) ||
		(btnEvent.Button == elevio.BT_Cab)))

}

func clearAtFloor(E Elevator) Elevator {
	println("clearing at floor")
	nextElevator := E
	nextElevator.Output.LocalOrders[E.Input.PrevFloor][elevio.BT_Cab] = false
	switch nextElevator.Output.MotorDirection {
	case elevio.MD_Up:
		if !OrdersAbove(nextElevator) && !nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallUp] {
			nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallDown] = false
		}
		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallUp] = false

	case elevio.MD_Down:
		if !OrdersBelow(nextElevator) && !nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallDown] {
			nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallUp] = false
		}
		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallDown] = false

	case elevio.MD_Stop:
		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallUp] = false
		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][elevio.BT_HallDown] = false

	}

	return nextElevator

}

func chooseDirection(E Elevator) DirectionStatePair {

	switch E.Output.MotorDirection {
	case elevio.MD_Up:

		if OrdersAbove(E) {
			return DirectionStatePair{elevio.MD_Up, MovingBetweenFloors}
		} else if OrdersHere(E) {
			return DirectionStatePair{elevio.MD_Stop, DoorOpen}

		} else if OrdersBelow(E) {
			return DirectionStatePair{elevio.MD_Down, MovingBetweenFloors}
		} else {
			return DirectionStatePair{elevio.MD_Stop, Idle}
		}
	case elevio.MD_Down:

		if OrdersBelow(E) {
			return DirectionStatePair{elevio.MD_Down, MovingBetweenFloors}
		} else if OrdersHere(E) {
			return DirectionStatePair{elevio.MD_Stop, DoorOpen}
		} else if OrdersAbove(E) {
			return DirectionStatePair{elevio.MD_Up, MovingBetweenFloors}
		} else {
			return DirectionStatePair{elevio.MD_Stop, Idle}
		}
	case elevio.MD_Stop:

		if OrdersHere(E) {

			return DirectionStatePair{elevio.MD_Stop, DoorOpen}
		} else if OrdersAbove(E) {

			return DirectionStatePair{elevio.MD_Up, MovingBetweenFloors}
		} else if OrdersBelow(E) {

			return DirectionStatePair{elevio.MD_Down, MovingBetweenFloors}
		} else {
			return DirectionStatePair{elevio.MD_Stop, Idle}
		}
	}
	return DirectionStatePair{elevio.MD_Stop, Idle}

}

func HandleNewOrder(order Order, E Elevator) Elevator {
	print("state at new order: ", E.State)
	nextElevator := E
	nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][order.ButtonEvent.Button] = true //legger inn den nye ordren.

	//først håndterer vi tilfellet der ordren er i etasjen vi er i.

	switch nextElevator.State {
	case DoorOpen:
		if shouldClearImmediately(nextElevator, order.ButtonEvent) {
			print("should clear immediately")
			nextElevator.Output.Door = true
			nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][order.ButtonEvent.Button] = false
			nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][elevio.BT_Cab] = false
			nextElevator.Output.MotorDirection = elevio.MD_Stop
			nextElevator.State = DoorOpen
			break
		}

	case Idle:
		println("idle at new order")
		DirectionStatePair := chooseDirection(nextElevator)

		nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		nextElevator.State = DirectionStatePair.State

	}

	print("state after new order: ", nextElevator.State)
	return nextElevator

}

func HandleFloorReached(event int, E Elevator) Elevator {
	nextElevator := E
	nextElevator.Input.PrevFloor = event
	switch nextElevator.State {
	case MovingBetweenFloors:
		println("moving between floors")
		if shouldStop(nextElevator) {
			println("should stop")
			nextElevator.Output.MotorDirection = elevio.MD_Stop
			nextElevator.Output.Door = true
			nextElevator = clearAtFloor(nextElevator)
			nextElevator.State = DoorOpen
		}

	}
	print("direction at floor reached: ", nextElevator.Output.MotorDirection)
	return nextElevator

}

func HandleDoorTimeout(E Elevator) Elevator {
	nextElevator := E
	switch nextElevator.State {
	case DoorOpen:
		DirectionStatePair := chooseDirection(nextElevator)
		nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		nextElevator.State = DirectionStatePair.State
		switch nextElevator.State {
		case DoorOpen:
			nextElevator = clearAtFloor(nextElevator)
		case Idle:
			nextElevator.Output.Door = false
			nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		}
	}
	println("state after door timeout", nextElevator.State)
	return nextElevator
}
