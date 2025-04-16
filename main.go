package main

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/peers"
	"Sanntid/pba"
	"os"
	"time"
)

var ID string
var startingAsPrimaryEnv string
var startingAsPrimary bool
var elevatorPortNumber string

func main() {
	peerTX := make(chan bool)
	peersRX := make(chan peers.PeerUpdate)
	primStatusRX := make(chan network.Status)
	nodeStatusTX := make(chan network.SingleElevatorStatus)
	RequestToPrimTX := make(chan network.Request)
	OrderFromPrimRX := make(chan network.Order)
	OrderCompletedTX := make(chan network.Request)
	LightUpdateFromPrimRX := make(chan network.LightUpdate)
	primaryMerge := make(chan network.Election)
	buttonPressCh := make(chan elevator.ButtonEvent)
	floorReachedCh := make(chan int)
	primaryRoutineDone := make(chan bool)
	backupRoutineDone := make(chan network.Takeover)

	ID = os.Getenv("ID")
	startingAsPrimaryEnv = os.Getenv("STARTASPRIM")
	if startingAsPrimaryEnv == "true" {
		startingAsPrimary = true
	} else {
		startingAsPrimary = false
	}
	elevatorPortNumber = os.Getenv("PORT")

	var E elevator.Elevator
	var prevLightMatrix [config.NFloors][config.NButtons]bool
	var NumRequests int = 1
	var lastOrderID int = 0
	var aloneOnNetwork = true
	var nodeStatusMap = make(map[string]network.SingleElevatorStatus)

	elevator.Init("localhost:"+elevatorPortNumber, config.NFloors)
	initializeElevator(E)
	E.State = elevator.DoorOpen
	E.Input.LocalRequests = [config.NFloors][config.NButtons]bool{}
	E.Output.LocalOrders = [config.NFloors][config.NButtons]bool{}
	E.Input.PrevFloor = elevator.GetFloor()
	E.DoorTimer = time.NewTimer(3 * time.Second)
	E.Output.Door = true
	E.DoorObstructed = elevator.GetObstruction()
	E.OrderCompleteTimer = time.NewTimer(config.OrderTimeout * time.Second)
	E.ObstructionTimer = time.NewTimer(7 * time.Second)
	E.ObstructionTimer.Stop()
	E.OrderCompleteTimer.Stop()

	go pba.PrimaryElection(ID, primaryMerge)
	initialPrimaryState := network.Takeover{
		StoredOrders:       [config.MElevators][config.NFloors][config.NButtons]bool{},
		PreviousPrimaryID:  "",
		Peerlist:           peers.PeerUpdate{},
		NodeMap:            nodeStatusMap,
		TakeOverInProgress: false,
	}
	if startingAsPrimary {
		go pba.Primary(ID, primaryMerge, initialPrimaryState, primaryRoutineDone)
	} else {
		go pba.Backup(ID, primaryMerge, backupRoutineDone)
	}

	go peers.Transmitter(12055, ID, peerTX)
	go peers.Receiver(12055, peersRX)
	go bcast.Transmitter(13057, RequestToPrimTX)
	go bcast.Receiver(13056, OrderFromPrimRX)
	go bcast.Transmitter(13058, OrderCompletedTX)
	go bcast.Transmitter(13059, nodeStatusTX)
	go bcast.Receiver(13060, LightUpdateFromPrimRX)
	go bcast.Receiver(13055, primStatusRX)
	go elevator.PollButtons(buttonPressCh)
	go elevator.PollFloorSensor(floorReachedCh)

	statusTicker := time.NewTicker(50 * time.Millisecond)

	for {
		select {
		case <-primaryRoutineDone:
			go pba.Backup(ID, primaryMerge, backupRoutineDone)

		case initialPrimaryState := <-backupRoutineDone:
			go pba.Primary(ID, primaryMerge, initialPrimaryState, primaryRoutineDone)

		case p := <-peersRX:
			aloneOnNetwork = len(p.Peers) == 0

		case lights := <-LightUpdateFromPrimRX:
			if (elevator.LightsDifferent(prevLightMatrix, lights.LightArray)) && lights.ID == ID {
				for i := range config.NButtons {
					for j := range config.NFloors {
						elevator.SetButtonLamp(elevator.ButtonType(i), j, lights.LightArray[j][i])
					}
				}
				prevLightMatrix = lights.LightArray
			}

		case btnEvent := <-buttonPressCh:
			E.Input.LocalRequests[btnEvent.Floor][btnEvent.Button] = true
			requestToPrimary := network.Request{
				ButtonEvent: btnEvent,
				ID:          ID,
				Orders:      E.Input.LocalRequests,
				RequestID:   NumRequests,
			}
			go network.SendRequestUpdate(RequestToPrimTX, requestToPrimary, config.IDToIndexMap)
			NumRequests++
			if aloneOnNetwork && btnEvent.Button == elevator.BT_Cab {
				offlineOrder := network.Order{ButtonEvent: btnEvent}
				E = elevator.HandleNewOrder(offlineOrder.ButtonEvent, E)
				elevator.SetButtonLamp(elevator.BT_Cab, btnEvent.Floor, true)
				setHardwareEffects(E)
			}

		case order := <-OrderFromPrimRX:
			if order.ResponisbleElevator != ID || order.ResponisbleElevator == ID && lastOrderID == order.OrderID {
				continue
			}
			E = elevator.HandleNewOrder(order.ButtonEvent, E)
			lastOrderID = order.OrderID
			setHardwareEffects(E)

		case a := <-floorReachedCh:
			E = elevator.HandleFloorReached(a, E)
			setHardwareEffects(E)
			if aloneOnNetwork {
				elevator.SetButtonLamp(elevator.BT_Cab, a, false)
			}

		case <-E.DoorTimer.C:
			E.DoorObstructed = elevator.GetObstruction()
			E = elevator.HandleDoorTimeout(E)
			setHardwareEffects(E)
			E.OrderCompleteTimer.Stop()
			for i := 0; i < len(E.Input.LastClearedButtons); i++ {
				orderMessage := network.Request{
					ButtonEvent: E.Input.LastClearedButtons[0],
					ID:          ID,
					Orders:      E.Output.LocalOrders,
					RequestID:   NumRequests}
				go network.SendRequestUpdate(OrderCompletedTX, orderMessage, config.IDToIndexMap)
				NumRequests++
				E.Input.LastClearedButtons = RemoveClearedOrder(E.Input.LastClearedButtons, E.Input.LastClearedButtons[0])
			}

		case <-E.OrderCompleteTimer.C:
			panic("Node failed to complete order, possible engine failure or faulty sensor")

		case <-E.ObstructionTimer.C:
			panic("Node failed to complete order, door obstruction")
		case <-statusTicker.C:
			nodeStatusTX <- network.SingleElevatorStatus{ID: ID, PrevFloor: E.Input.PrevFloor, MotorDirection: E.Output.MotorDirection, Orders: E.Output.LocalOrders}

		}
	}
}

func setHardwareEffects(E elevator.Elevator) {
	elevator.SetMotorDirection(E.Output.MotorDirection)
	elevator.SetDoorOpenLamp(E.Output.Door)
	elevator.SetFloorIndicator(E.Input.PrevFloor)

}

func RemoveClearedOrder(clearedOrders []elevator.ButtonEvent, event elevator.ButtonEvent) []elevator.ButtonEvent {
	var remainingOrders []elevator.ButtonEvent
	for i := 0; i < len(clearedOrders); i++ {
		if clearedOrders[i] != event {
			remainingOrders = append(remainingOrders, clearedOrders[i])
		}
	}
	return remainingOrders
}

func initializeElevator(E elevator.Elevator) {
	for j := 0; j < 4; j++ {
		elevator.SetButtonLamp(elevator.BT_HallUp, j, false)
		elevator.SetButtonLamp(elevator.BT_HallDown, j, false)
		elevator.SetButtonLamp(elevator.BT_Cab, j, false)
	}

	E.ObstructionTimer = time.NewTimer(config.ObstructionTimeout * time.Second)
	for {
		elevator.SetMotorDirection(elevator.MD_Up)
		if elevator.GetFloor() != -1 {
			elevator.SetMotorDirection(elevator.MD_Stop)
			break
		}
		time.Sleep(config.OrderTimeout * time.Millisecond)
	}
	if E.DoorObstructed { // If elevator starts with an obstructed door, it will start in a failure state and does not broadcast alive status.
		for {
			if !elevator.GetObstruction() {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		E.ObstructionTimer.Stop()
	}

}
