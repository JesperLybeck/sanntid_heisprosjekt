package fsm

import (
	"Sanntid/elevio"
	"fmt"
	"strconv"
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
	LocalOrders        [NFloors][NButtons]bool
}

type ElevatorInput struct {
	LocalRequests [NFloors][NButtons]bool
	PrevFloor     int
}
type Elevator struct {
	State              ElevatorState
	Input              ElevatorInput
	Output             ElevatorOutput
	DoorTimer          *time.Timer
	OrderCompleteTimer *time.Timer
	ObstructionTimer   *time.Timer
	DoorObstructed     bool
}

type DirectionStatePair struct {
	Direction elevio.MotorDirection
	State     ElevatorState
}

func OrdersAbove(E Elevator) bool {

	for i := E.Input.PrevFloor + 1; i < NFloors; i++ {
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
		(btnEvent.Button == elevio.BT_Cab)) && !(E.State == DoorOpen))

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

func HandleNewOrder(order Order, E Elevator) Elevator {
	//print("handling new order")
	wasIdleAtNewOrder := E.State == Idle
	nextElevator := E
	nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][order.ButtonEvent.Button] = true //legger inn den nye ordren.

	//først håndterer vi tilfellet der ordren er i etasjen vi er i.

	switch nextElevator.State {

	case DoorOpen:
		if shouldClearImmediately(nextElevator, order.ButtonEvent) {
			//uten disse, vil heisen stå i 6 sekunder.
			nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][order.ButtonEvent.Button] = false
			nextElevator.Output.LocalOrders[order.ButtonEvent.Floor][elevio.BT_Cab] = false
			nextElevator.Output.MotorDirection = elevio.MD_Stop
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
			nextElevator = clearAtFloor(nextElevator)
		}

		nextElevator.Output.MotorDirection = DirectionStatePair.Direction
		nextElevator.State = DirectionStatePair.State

	}
	if nextElevator.Output.MotorDirection != elevio.MD_Stop && wasIdleAtNewOrder {

		nextElevator.OrderCompleteTimer.Stop() // Stop before reset to ensure clean state
		nextElevator.OrderCompleteTimer.Reset(OrderTimeout * time.Second)
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
			fmt.Println("Resetting door timer")
			nextElevator.ObstructionTimer.Reset(7 * time.Second)

			nextElevator.State = DoorOpen

		}
		nextElevator.OrderCompleteTimer.Reset(OrderTimeout * time.Second)

	}

	return nextElevator

}
func LightsDifferent(lightArray1 [NFloors][NButtons]bool, lightArray2 [NFloors][NButtons]bool) bool {
	for i := 0; i < NFloors; i++ {
		for j := 0; j < NButtons; j++ {
			if lightArray1[i][j] != lightArray2[i][j] {
				return true
			}
		}
	}
	return false
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

		nextElevator.OrderCompleteTimer.Reset(OrderTimeout * time.Second)
	}
	if nextElevator.DoorObstructed {
		print("Door obstructed")
		nextElevator.Output.Door = true
		nextElevator.State = DoorOpen
		nextElevator.Output.MotorDirection = elevio.MD_Stop
		nextElevator.DoorTimer.Reset(4 * time.Second)
		nextElevator.OrderCompleteTimer.Reset(OrderTimeout * time.Second)

	} else {
		fmt.Println("Door timer stopped")
		nextElevator.ObstructionTimer.Stop()
		nextElevator.Output.Door = false
	}

	return nextElevator
}

// vi kan kalle disse som go routines der vi sender requests, til prim, og bekrefter utførte ordre.
func SendRequestUpdate(transmitterChan chan<- Request, ackChan <-chan Status, message Request, requestID int) {

	sendingTicker := time.NewTicker(30 * time.Millisecond)
	messageTimer := time.NewTimer(5 * time.Second)

	defer sendingTicker.Stop()

	//dette betyr at andre noder kan acke ordre som ikke er til dem?

	messagesSent := 0

	for {
		select {
		case <-sendingTicker.C:

			transmitterChan <- message
			messagesSent++

		case status := <-ackChan: //kan dette skje på samme melding?

			floor := message.ButtonEvent.Floor
			button := message.ButtonEvent.Button
			print("ID: ", message.ID, "index: ", IpToIndexMap[message.ID])
			for j := 0; j < MElevators; j++ {
				if (status.Orders[j][floor][button] == message.Orders[floor][button]) && messagesSent > 0 {

					return
				}
			}

		case <-messageTimer.C:
			print("No ack received for request, stopping transmission.")
			//vi trenger ikke å sende error her. vi kan anta bruker trykker på knappen på nytt.
			return

		}
	}
}

func SendOrder(transmitterChan chan<- Order, ackChan <-chan SingleElevatorStatus, message Order, ID string, OrderID int, ResendChan chan<- Request){
	messageTimer := time.NewTimer(5 * time.Second)
	sendingTicker := time.NewTicker(30 * time.Millisecond)

	defer sendingTicker.Stop()
	messagesSent := 0
	// er vi nødt til å acke ordre gitt i etasje vi allerede er i?
	for {
		select {
		case <-sendingTicker.C:
			messagesSent++
			transmitterChan <- message
		case status := <-ackChan:
			button := message.ButtonEvent.Button
			floor := message.ButtonEvent.Floor

			if message.ResponisbleElevator == status.ID && (status.Orders[floor][button] || (message.ButtonEvent.Floor == status.PrevFloor && messagesSent > 0)) {
				return
			}
		case <-messageTimer.C:
			RequestID := message.OrderID
			Reassign := Request{ID: ID, ButtonEvent: message.ButtonEvent, Orders: NodeStatusMap[ID].Orders, RequestID: RequestID}
			ResendChan <- Reassign
			return 
			
			//kan vi throwe en error her, som sørger for at ordren forsøkes håndtert på nytt? den kan da sendes til en annen node i stedet??

			
		}
	}
}

func incrementMessage(messageID string) string {
	nodeID := messageID[:3]
	messageNumber := messageID[3:]
	messageNumberInt, _ := strconv.Atoi(messageNumber)
	messageNumberInt++
	messageNumber = strconv.Itoa(messageNumberInt)
	return nodeID + messageNumber

}
