package elevator

import (
	"Sanntid/config"
	//"Sanntid/network"
	"fmt"
	"time"
)

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
	case MD_Down:
		return (E.Output.LocalOrders[E.Input.PrevFloor][BT_HallDown] ||
			E.Output.LocalOrders[E.Input.PrevFloor][BT_Cab] ||
			!OrdersBelow(E))
	case MD_Up:

		return (E.Output.LocalOrders[E.Input.PrevFloor][BT_HallUp] ||
			E.Output.LocalOrders[E.Input.PrevFloor][BT_Cab] ||
			!OrdersAbove(E))
	case MD_Stop:
		return true

	}
	return true

}

// checking if the order is to be cleared immediately.
func shouldClearImmediately(E Elevator, btnEvent ButtonEvent) bool {

	return ((E.Input.PrevFloor == btnEvent.Floor) && ((E.Output.MotorDirection == MD_Up && btnEvent.Button == BT_HallUp) ||
		(E.Output.MotorDirection == MD_Down && btnEvent.Button == BT_HallDown) ||
		(E.Output.MotorDirection == MD_Stop) ||
		(btnEvent.Button == BT_Cab)) && !(E.State == DoorOpen))

}

func ClearAtFloor(E Elevator) Elevator {

	nextElevator := E
	nextElevator.Output.LocalOrders[E.Input.PrevFloor][BT_Cab] = false

	switch nextElevator.Output.MotorDirection {
	case MD_Up:
		if !OrdersAbove(nextElevator) && !nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallUp] {
			nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallDown] = false
		}
		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallUp] = false

	case MD_Down:
		if !OrdersBelow(nextElevator) && !nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallDown] {
			nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallUp] = false
		}
		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallDown] = false

	case MD_Stop:

		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallUp] = false
		nextElevator.Output.LocalOrders[nextElevator.Input.PrevFloor][BT_HallDown] = false

	}

	return nextElevator

}

func chooseDirection(E Elevator) DirectionStatePair {

	switch E.Output.prevMotorDirection {
	case MD_Up:

		if OrdersAbove(E) {
			return DirectionStatePair{MD_Up, MovingBetweenFloors}
		} else if OrdersHere(E) {
			return DirectionStatePair{MD_Stop, DoorOpen}

		} else if OrdersBelow(E) {
			if WasHallUp(E.Input.LastClearedButtons) {
				print("-------------It was hall up, but we are are changing directions. Stopping 3 more second-------------")
				return DirectionStatePair{MD_Stop, DoorOpen} //spesialcase ekstra dørtimer
			} else {
				return DirectionStatePair{MD_Down, MovingBetweenFloors}
			}
		} else {
			return DirectionStatePair{MD_Stop, Idle}
		}
	case MD_Down:

		if OrdersBelow(E) {
			return DirectionStatePair{MD_Down, MovingBetweenFloors}
		} else if OrdersHere(E) {
			return DirectionStatePair{MD_Stop, DoorOpen}
		} else if OrdersAbove(E) {
			if WasHallDown(E.Input.LastClearedButtons) {
				print("---------------It was hall Down, but we are are changing directions. Stopping 3 more second-------------")
				return DirectionStatePair{MD_Stop, DoorOpen} //spesialcase ekstra dørtimer
			} else {
				return DirectionStatePair{MD_Up, MovingBetweenFloors}
			}
		} else {
			return DirectionStatePair{MD_Stop, Idle}
		}
	case MD_Stop:

		if OrdersHere(E) {

			return DirectionStatePair{MD_Stop, DoorOpen}
		} else if OrdersAbove(E) {

			return DirectionStatePair{MD_Up, MovingBetweenFloors}
		} else if OrdersBelow(E) {

			return DirectionStatePair{MD_Down, MovingBetweenFloors}
		} else {
			return DirectionStatePair{MD_Stop, Idle}
		}
	}
	return DirectionStatePair{MD_Stop, Idle}

}

func LastClearedButtons(e Elevator, b Elevator) []ButtonEvent {
	lcb := []ButtonEvent{}
	order1 := e.Output.LocalOrders
	order2 := b.Output.LocalOrders
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons; j++ {
			if order1[i][j] != order2[i][j] {
				lcb = append(lcb, ButtonEvent{Floor: i, Button: ButtonType(j)})
			}
		}
	}
	return lcb
}

func HandleNewOrder(order ButtonEvent, E Elevator) Elevator {
	wasIdleAtNewOrder := E.State == Idle
	nextElevator := E
	nextElevator.Output.LocalOrders[order.Floor][order.Button] = true //legger inn den nye ordren.

	//først håndterer vi tilfellet der ordren er i etasjen vi er i.

	switch nextElevator.State {

	case DoorOpen:
		if shouldClearImmediately(nextElevator, order) {
			//uten disse, vil heisen stå i 6 sekunder.
			nextElevator.Output.LocalOrders[order.Floor][order.Button] = false
			nextElevator.Output.LocalOrders[order.Floor][BT_Cab] = false
			nextElevator.Output.MotorDirection = MD_Stop
			nextElevator.State = DoorOpen
			nextElevator.Output.Door = true
			nextElevator.DoorTimer.Reset(3 * time.Second)
			nextElevator.ObstructionTimer.Reset(7 * time.Second)
			print("Clearing order immediately, resetting obstruction timer")
			//her returnerer vi tomt, så prim som venter på ack, får aldri denne.

			break
		}

	case Idle:

		nextElevator.Output.prevMotorDirection = nextElevator.Output.MotorDirection
		DirectionStatePair := chooseDirection(nextElevator)
		if DirectionStatePair.State == DoorOpen {
			nextElevator.ObstructionTimer.Reset(7 * time.Second)
			print("Clearing order immediately, resetting obstruction timer")
			nextElevator.Output.Door = true
			nextElevator.DoorTimer.Reset(3 * time.Second)
			nextElevator = ClearAtFloor(nextElevator)
		}

		nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		nextElevator.State = DirectionStatePair.State

	}
	if nextElevator.Output.MotorDirection != MD_Stop && wasIdleAtNewOrder {

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
			nextElevator = ClearAtFloor(nextElevator)
			nextElevator.Input.LastClearedButtons = LastClearedButtons(model, nextElevator)
			nextElevator.Output.MotorDirection = MD_Stop
			nextElevator.Output.Door = true

			nextElevator.DoorTimer.Reset(3 * time.Second)
			fmt.Println("Resetting door timer")
			nextElevator.ObstructionTimer.Reset(7 * time.Second)

			nextElevator.State = DoorOpen

		}
		nextElevator.OrderCompleteTimer.Reset(config.OrderTimeout * time.Second)

	}
	return nextElevator
}

func HandleDoorTimeout(E Elevator) Elevator {

	nextElevator := E
	switch nextElevator.State {
	// Er ikke alltid døren åpen når den timer ut?
	case DoorOpen:

		DirectionStatePair := chooseDirection(nextElevator)
		nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		nextElevator.State = DirectionStatePair.State
		switch nextElevator.State {
		case DoorOpen:
			nextElevator.Output.Door = true
			nextElevator.DoorTimer.Reset(3 * time.Second)
			nextElevator = ClearAtFloor(nextElevator)
			//men door open til door open):
		case Idle:
			nextElevator.Output.Door = false
			nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		case MovingBetweenFloors:
			nextElevator.Output.Door = false
		}

	}
	if nextElevator.Output.MotorDirection != MD_Stop {

		nextElevator.OrderCompleteTimer.Reset(config.OrderTimeout * time.Second)
	}
	if nextElevator.DoorObstructed {
		print("Door obstructed")
		nextElevator.Output.Door = true
		nextElevator.State = DoorOpen
		nextElevator.Output.MotorDirection = MD_Stop
		nextElevator.DoorTimer.Reset(4 * time.Second)
		nextElevator.OrderCompleteTimer.Reset(config.OrderTimeout * time.Second)

	} else {
		fmt.Println("Door timer stopped")
		nextElevator.ObstructionTimer.Stop()
		nextElevator.Output.Door = false
	}

	return nextElevator
}

func LightsDifferent(lightArray1 [config.NFloors][config.NButtons]bool, lightArray2 [config.NFloors][config.NButtons]bool) bool {
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons; j++ {
			if lightArray1[i][j] != lightArray2[i][j] {
				return true
			}
		}
	}
	return false
}

func WasHallUp(buttonArray []ButtonEvent) bool {
	for i := 0; i < len(buttonArray); i++ {
		if buttonArray[i].Button == BT_HallUp {
			return true
		}
	}
	return false
}

func WasHallDown(buttonArray []ButtonEvent) bool {
	for i := 0; i < len(buttonArray); i++ {
		if buttonArray[i].Button == BT_HallDown {
			return true
		}
	}
	return false
}
