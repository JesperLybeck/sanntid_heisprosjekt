package main

import (
	"Sanntid/config"
	"Sanntid/elevator"

	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/localip"
	"Sanntid/networkDriver/peers"
	"Sanntid/pba"

	"os"
	"time"
)

// ------------------------------Utility functions--------------------------------

func setHardwareEffects(E elevator.Elevator) {
	elevator.SetMotorDirection(E.Output.MotorDirection)
	elevator.SetDoorOpenLamp(E.Output.Door)
	elevator.SetFloorIndicator(E.Input.PrevFloor)

}

// ----------------------------ENVIRONMENT VARIABLES-----------------------------
var ID string
var startingAsPrimaryEnv string
var startingAsPrimary bool
var elevatorPortNumber string

func main() {
	//-----------------------------CHANNELS-----------------------------
	peerTX := make(chan bool)
	nodeStatusTX := make(chan network.SingleElevatorStatus) //strictly having both should be unnecessary.
	RequestToPrimTX := make(chan network.Request)
	OrderFromPrimRX := make(chan network.Order)
	OrderCompletedTX := make(chan network.Request)
	LightUpdateFromPrimRX := make(chan network.LightUpdate)
	peersRX := make(chan peers.PeerUpdate)
	primStatusRX := make(chan network.Status)

	buttonPressCh := make(chan elevator.ButtonEvent)
	floorReachedCh := make(chan int)

	primaryMerge := make(chan network.Election)

	aloneOnNetwork := true

	//doorTimeoutCh := make(chan bool)

	//vi skiller mellom intern komunikasjon og kommunikasjon med primary.
	//channels for internal communication is denoted with "eventCh" channels for external comms are named "EventTX or RX"

	//-----------------------------TIMERS-----------------------------
	statusTicker := time.NewTicker(100 * time.Millisecond)

	//------------------------ASSIGNING ENVIRONMENT VARIABLES------------------------
	ID = os.Getenv("ID")
	if ID == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			localIP = "DISCONNECTED"
		}
		ID = localIP
	}

	startingAsPrimaryEnv = os.Getenv("STARTASPRIM")
	if startingAsPrimaryEnv == "true" {
		startingAsPrimary = true
	} else {
		startingAsPrimary = false
	}

	elevatorPortNumber = os.Getenv("PORT") // Read the environment variable
	if elevatorPortNumber == "" {
		elevatorPortNumber = "15657" // Default value if the environment variable is not set
	}
	//-----------------------------STATE MACHINE VARIABLES-----------------------------
	var E elevator.Elevator
	var prevLightMatrix [config.NFloors][config.NButtons]bool
	var NumRequests int = 1
	var lastOrderID int = 0

	//-----------------------------INITIALIZING ELEVATOR-----------------------------
	elevator.Init("localhost:"+elevatorPortNumber, config.NFloors) //when starting, the elevator goes up until it reaches a floor.

	for {
		elevator.SetMotorDirection(elevator.MD_Up)
		if elevator.GetFloor() != -1 {
			elevator.SetMotorDirection(elevator.MD_Stop)
			break
		}
		time.Sleep(config.OrderTimeout * time.Millisecond) // Add small delay between polls
	}
	for j := 0; j < 4; j++ {
		elevator.SetButtonLamp(elevator.BT_HallUp, j, false) //skrur av alle lys ved initsialisering. Nødvendig???
		elevator.SetButtonLamp(elevator.BT_HallDown, j, false)
		elevator.SetButtonLamp(elevator.BT_Cab, j, false)
	}
	//elevator state machine variables are initialized.
	E.State = elevator.Idle                                         //after initializing the elevator, it goes to the idle state.
	E.Input.LocalRequests = [config.NFloors][config.NButtons]bool{} // not strictly necessary, but...
	E.Output.LocalOrders = [config.NFloors][config.NButtons]bool{}
	E.Input.PrevFloor = elevator.GetFloor()
	E.DoorTimer = time.NewTimer(3 * time.Second)
	E.Output.Door = true
	E.DoorObstructed = elevator.GetObstruction()
	E.OrderCompleteTimer = time.NewTimer(config.OrderTimeout * time.Second)
	E.ObstructionTimer = time.NewTimer(7 * time.Second)

	if E.DoorObstructed {
		for {
			if !elevator.GetObstruction() {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		E.ObstructionTimer.Stop()
	}

	E.OrderCompleteTimer.Stop()
	//-----------------------------GO ROUTINES-----------------------------
	go pba.RoleElection(ID, primaryMerge)

	initialPrimaryState := network.Takeover{
		StoredOrders:       [config.MElevators][config.NFloors][config.NButtons]bool{},
		PreviousPrimaryID:  "",
		Peerlist:           peers.PeerUpdate{},
		NodeMap:            map[string]network.SingleElevatorStatus{},
		TakeOverInProgress: false,
	}
	//go pba.Primary(ID, primaryMerge) //starting go routines for primary and backup.
	primaryRoutineDone := make(chan bool)
	backupRoutineDone := make(chan network.Takeover)

	if startingAsPrimary {
		go pba.Primary(ID, primaryMerge, initialPrimaryState, primaryRoutineDone)

	} else {

		go pba.Backup(ID, primaryMerge, backupRoutineDone)

	}

	go elevator.PollButtons(buttonPressCh) //starting go routines for polling HW
	go elevator.PollFloorSensor(floorReachedCh)

	go bcast.Transmitter(13057, RequestToPrimTX) //starting go routines for network communication with other primary.
	go bcast.Receiver(13056, OrderFromPrimRX)
	go bcast.Transmitter(13058, OrderCompletedTX)
	go bcast.Transmitter(13059, nodeStatusTX)
	go bcast.Receiver(13060, LightUpdateFromPrimRX)
	go bcast.Receiver(13055, primStatusRX)
	go peers.Transmitter(12055, ID, peerTX)
	go peers.Receiver(12055, peersRX)

	//-----------------------------MAIN LOOP-----------------------------
	for {

		select {

		case <-primaryRoutineDone:

			print("demoting to backup")
			go pba.Backup(ID, primaryMerge, backupRoutineDone)
			//hvis noden er ferdig som prim (har blitt nedgradert)
		case initialPrimaryState := <-backupRoutineDone:
			print("promoting to primary")
			//hvis noden er ferdig som backup (har blitt oppgradert)
			go pba.Primary(ID, primaryMerge, initialPrimaryState, primaryRoutineDone)
		case p := <-peersRX: //vi klarer oss vell uten denne, med casen over?
			// To register if alone on network and enter offline mode
			aloneOnNetwork = len(p.Peers) == 0 /*
				if len(p.Peers) == 0 {
					pba.AloneOnNetwork = true
				} else {
					pba.AloneOnNetwork = false
				}*/
		case lights := <-LightUpdateFromPrimRX:

			//when light update is received from primary, the node updates its own lights with the newest information.
			if (elevator.LightsDifferent(prevLightMatrix, lights.LightArray)) && lights.ID == ID {

				for i := range config.NButtons {
					for j := range config.NFloors {
						elevator.SetButtonLamp(elevator.ButtonType(i), j, lights.LightArray[j][i]) // vi kan vurdere om denne faktisk kan pakkes inn i en funksjon da vi gjør dette flere steder  koden.

					}

				}
				prevLightMatrix = lights.LightArray

			}

		case btnEvent := <-buttonPressCh: //case for å håndtere knappe trykk. Sender ordre til prim.			// Hvis heisen er i etasje n og får knappetrykk i n trenger man ikke å sende ordre til primary
			E.Input.LocalRequests[btnEvent.Floor][btnEvent.Button] = true
			requestToPrimary := network.Request{
				ButtonEvent: btnEvent,
				ID:          ID,
				Orders:      E.Input.LocalRequests,
				RequestID:   NumRequests,
			}

			// ISSUE! when the order is delegated to a different node, we cant ack on
			context := "called by buttonPress by " + string(ID)
			go network.SendRequestUpdate(RequestToPrimTX, requestToPrimary, config.IDToIndexMap, context)
			NumRequests++

			if aloneOnNetwork && btnEvent.Button == elevator.BT_Cab {
				offlineOrder := network.Order{ButtonEvent: btnEvent, ResponisbleElevator: ID}
				E = elevator.HandleNewOrder(offlineOrder.ButtonEvent, E) //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.
				elevator.SetButtonLamp(elevator.BT_Cab, btnEvent.Floor, true)
				setHardwareEffects(E)

			}

			//vi diskuterte om vi trengte å ha egen case for cab. Vi kom frem til at det ikke trengs fordi:
			//i: Hvis noden har kræsjet, tar vi ikke ordre.
			//ii: Hvis noden er uten nett, setter den seg selv til primary, den vil dermed ta ordre fra cab selv.

		case order := <-OrderFromPrimRX: // I denne casen mottar noden en ordre fra primary.

			if order.ResponisbleElevator != ID || order.ResponisbleElevator == ID && lastOrderID == order.OrderID { // hvis ordren er til en annen heis, ignorer.
				//denne kunne også strengt tatt gått inn i handle new Order functionen.
				//fmt.Print("anti spam order received from prim", order.ResponisbleElevator != ID, "------------------>", order.ResponisbleElevator, "<---->", ID)
				continue
			}

			//problem om heisen allerede er i etasjen ordren er i.
			//Da vil primary ikke få ack, fordi handle new order legger ikke til ordren i localOrders.
			//programmet terminerer ikke.
			E = elevator.HandleNewOrder(order.ButtonEvent, E)
			lastOrderID = order.OrderID //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.

			setHardwareEffects(E)

		case a := <-floorReachedCh:

			E = elevator.HandleFloorReached(a, E)

			setHardwareEffects(E)

			if aloneOnNetwork {
				elevator.SetButtonLamp(elevator.BT_Cab, a, false)
			}

		case <-E.DoorTimer.C:

			print("door timer expired")

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
				contect := "called by doorTimer by " + string(ID)
				go network.SendRequestUpdate(OrderCompletedTX, orderMessage, config.IDToIndexMap, contect)
				NumRequests++
				E.Input.LastClearedButtons = RemoveClearedOrder(E.Input.LastClearedButtons, E.Input.LastClearedButtons[0])
			}

		case <-E.OrderCompleteTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, possible engine failure or faulty sensor")
		case <-E.ObstructionTimer.C:

			panic("Node failed to complete order, door obstruction")
		case <-statusTicker.C:

			nodeStatusTX <- network.SingleElevatorStatus{ID: ID, PrevFloor: E.Input.PrevFloor, MotorDirection: E.Output.MotorDirection, Orders: E.Output.LocalOrders}

		}

	}

}

// denne må flyttes!!!!
func RemoveClearedOrder(clearedOrders []elevator.ButtonEvent, event elevator.ButtonEvent) []elevator.ButtonEvent {
	var remainingOrders []elevator.ButtonEvent
	for i := 0; i < len(clearedOrders); i++ {
		if clearedOrders[i] != event {
			remainingOrders = append(remainingOrders, clearedOrders[i])
		}
	}
	return remainingOrders
}
