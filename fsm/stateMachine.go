package fsm

import (
	"Sanntid/config"
	"Sanntid/elevio"
	"time"
	
)

type ElevatorState int

const (
	Idle ElevatorState = iota
	MovingBetweenFloors
	MovingPassingFloor
	DoorOpen
)

type ElevatorOutput struct {
	MotorDirection     elevio.MotorDirection
	prevMotorDirection elevio.MotorDirection
	Door               bool
	LocalOrders        [config.NFloors][config.NButtons]bool
}

type ElevatorInput struct {
	LocalRequests [config.NFloors][config.NButtons]bool
	PrevFloor     int
}
type Elevator struct {
	State              ElevatorState
	Input              ElevatorInput
	Output             ElevatorOutput
	DoorTimer          *time.Timer
	OrderCompleteTimer *time.Timer
	DoorObstructed     bool
}

type DirectionStatePair struct {
	Direction elevio.MotorDirection
	State     ElevatorState
}

func OrdersAbove(E Elevator) bool {

	for i := E.Input.PrevFloor + 1; i < config.NFloors; i++ {
		if E.Output.LocalOrders[i][0] || E.Output.LocalOrders[i][1] || E.Output.LocalOrders[i][2] {

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
	for i := 0; i < config.NButtons; i++ {
		for j := 0; j < config.NFloors; j++ {
			if queue[j][i] {
				return false
			}
		}
	}
	return true
}

func shouldStop(E Elevator) bool {

	switch E.Output.MotorDirection {
	case elevio.MD_Down:
		return (E.Output.LocalOrders[E.Input.PrevFloor][elevio.BT_HallDown] ||
			E.Output.LocalOrders[E.Input.PrevFloor][elevio.BT_Cab] ||
			!OrdersBelow(E))
	case elevio.MD_Up:

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

	return ((E.Input.PrevFloor == btnEvent.Floor) && ((E.Output.MotorDirection == elevio.MD_Up && btnEvent.Button == elevio.BT_HallUp) ||
		(E.Output.MotorDirection == elevio.MD_Down && btnEvent.Button == elevio.BT_HallDown) ||
		(E.Output.MotorDirection == elevio.MD_Stop) ||
		(btnEvent.Button == elevio.BT_Cab)))

}

func clearAtFloor(E Elevator) Elevator {

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

	switch E.Output.prevMotorDirection {
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

func HandleNewOrder(order config.Order, E Elevator) Elevator {
	wasIdleAtNewOrder := E.State == Idle
	nextElevator := E
	nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][order.ButtonEvent.Button] = true //legger inn den nye ordren.

	//først håndterer vi tilfellet der ordren er i etasjen vi er i.

	switch nextElevator.State {
	case DoorOpen:
		if shouldClearImmediately(nextElevator, order.ButtonEvent) {

			nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][order.ButtonEvent.Button] = false
			nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][elevio.BT_Cab] = false
			nextElevator.Output.MotorDirection = elevio.MD_Stop
			nextElevator.State = DoorOpen
			nextElevator.Output.Door = true
			nextElevator.DoorTimer.Reset(3 * time.Second)

			break
		}

	case Idle:

		nextElevator.Output.prevMotorDirection = nextElevator.Output.MotorDirection
		DirectionStatePair := chooseDirection(nextElevator)
		if DirectionStatePair.State == DoorOpen {
			nextElevator.Output.Door = true
			nextElevator.DoorTimer.Reset(3 * time.Second)
			nextElevator = clearAtFloor(nextElevator)
		}

		nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		nextElevator.State = DirectionStatePair.State

	}
	if nextElevator.Output.MotorDirection != elevio.MD_Stop && wasIdleAtNewOrder {
		print("resetting order complete timer")
		nextElevator.OrderCompleteTimer.Stop() // Stop before reset to ensure clean state
		nextElevator.OrderCompleteTimer.Reset(config.OrderTimeout * time.Second)
	}
	return nextElevator

}

func HandleFloorReached(event int, E Elevator) Elevator {

	model := E
	model.Input.PrevFloor = event
	nextElevator := E
	nextElevator.Input.PrevFloor = event
	switch nextElevator.State {
	case MovingBetweenFloors:

		if shouldStop(nextElevator) {

			nextElevator.Output.prevMotorDirection = nextElevator.Output.MotorDirection
			nextElevator = clearAtFloor(nextElevator)
			nextElevator.Output.MotorDirection = elevio.MD_Stop
			nextElevator.Output.Door = true

			nextElevator.DoorTimer.Reset(3 * time.Second)

			nextElevator.State = DoorOpen
		}
		nextElevator.OrderCompleteTimer.Reset(config.OrderTimeout * time.Second)

	}

	return nextElevator

}

func HandleDoorTimeout(E Elevator) Elevator {
	print("handle door timeout")
	nextElevator := E
	switch nextElevator.State {
	case DoorOpen:

		DirectionStatePair := chooseDirection(nextElevator)
		nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		nextElevator.State = DirectionStatePair.State
		switch nextElevator.State {
		case DoorOpen:
			nextElevator.Output.Door = true
			nextElevator.DoorTimer.Reset(3 * time.Second)
			nextElevator = clearAtFloor(nextElevator)
			//men door open til door open):
		case Idle:
			nextElevator.Output.Door = false
			nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		case MovingBetweenFloors:
			nextElevator.Output.Door = false
		}
	}
	if nextElevator.Output.MotorDirection != elevio.MD_Stop {
		print("resetting order complete timer")
		nextElevator.OrderCompleteTimer.Reset(config.OrderTimeout * time.Second)
	}
	if nextElevator.State != DoorOpen && nextElevator.DoorObstructed {
		print("door obstructed")
		nextElevator.Output.Door = true
		nextElevator.State = DoorOpen
		nextElevator.DoorTimer.Reset(4 * time.Second)
		nextElevator.OrderCompleteTimer.Reset(config.OrderTimeout * time.Second)
	}

	return nextElevator
}
