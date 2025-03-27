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
	print("Setting hardware effects")
	print("------",E.Output.Door,"-------")
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
	var AloneOnNetwork bool
	//-----------------------------CHANNELS-----------------------------
	peerTX := make(chan bool)
	nodeStatusTX := make(chan network.SingleElevatorStatus) //strictly having both should be unnecessary.
	RequestToPrimTX := make(chan network.Request)
	OrderFromPrimRX := make(chan network.Order)
	OrderCompletedTX := make(chan network.Request)
	LightUpdateFromPrimRX := make(chan network.LightUpdate)
	peersRX := make(chan peers.PeerUpdate)

	buttonPressCh := make(chan elevator.ButtonEvent)
	floorReachedCh := make(chan int)

	primaryMerge := make(chan network.Election)
	primaryTakeover := make(chan network.Takeover)

	activateBackup := make(chan bool)
	startRoleElection := make(chan bool)

	//doorTimeoutCh := make(chan bool)

	//vi skiller mellom intern komunikasjon og kommunikasjon med primary.
	//channels for internal communication is denoted with "eventCh" channels for external comms are named "EventTX or RX"

	//-----------------------------TIMERS-----------------------------
	statusTicker := time.NewTicker(30 * time.Millisecond)

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
	var lastClearedButtons []elevator.ButtonEvent

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
	E.OrderCompleteTimer = time.NewTimer(config.OrderTimeout * time.Second)
	E.ObstructionTimer = time.NewTimer(7 * time.Second)

	E.ObstructionTimer.Stop()

	E.OrderCompleteTimer.Stop()
	//-----------------------------GO ROUTINES-----------------------------
	go pba.RoleElection(ID, activateBackup, startRoleElection, primaryMerge)
	go pba.Primary(ID, activateBackup, startRoleElection, primaryMerge, primaryTakeover) //starting go routines for primary and backup.
	go pba.Backup(ID, activateBackup, primaryTakeover)

	if startingAsPrimary {
		time.Sleep(500 * time.Millisecond)
		primaryTakeover <- network.Takeover{TakeOverInProgress: false, NodeStatusMap: make(map[string]network.SingleElevatorStatus)}
	} else {
		activateBackup <- true
	}

	go elevator.PollButtons(buttonPressCh) //starting go routines for polling HW
	go elevator.PollFloorSensor(floorReachedCh)

	go bcast.Transmitter(13057, RequestToPrimTX) //starting go routines for network communication with other primary.
	go bcast.Receiver(13056, OrderFromPrimRX)
	go bcast.Transmitter(13058, OrderCompletedTX)
	go bcast.Transmitter(13059, nodeStatusTX)
	go bcast.Receiver(13060, LightUpdateFromPrimRX)
	go peers.Transmitter(12055, ID, peerTX)
	go peers.Receiver(12055, peersRX)

	//-----------------------------MAIN LOOP-----------------------------
	for {

		select {
		/*case primStatus := <-primStatusRX:
		//Hva om vi gjør en del av dette her, heller enn i pba?
		fsm.IpToIndexMap = primStatus.Map
		fsm.LatestPeerList = primStatus.Peerlist	*/

		case p := <-peersRX: //vi klarer oss vell uten denne, med casen over?
			// To register if alone on network and enter offline mode
			if len(p.Peers) == 0 {
				AloneOnNetwork = true
			} else {
				AloneOnNetwork = false
			}
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

			go network.SendRequestUpdate(RequestToPrimTX, requestToPrimary, NumRequests)
			NumRequests++

			if AloneOnNetwork && btnEvent.Button == elevator.BT_Cab {
				offlineOrder := network.Order{ButtonEvent: btnEvent, ResponisbleElevator: ID, OrderID: 1}
				E = elevator.HandleNewOrder(offlineOrder.ButtonEvent, E) //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.
				elevator.SetButtonLamp(elevator.BT_Cab, btnEvent.Floor, true)
				setHardwareEffects(E)
				print("Offline order received")
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

			if AloneOnNetwork {
				elevator.SetButtonLamp(elevator.BT_Cab, a, false)
			}

		case <-E.DoorTimer.C:

			E.DoorObstructed = elevator.GetObstruction()

			E = elevator.HandleDoorTimeout(E)

			setHardwareEffects(E)

			E.OrderCompleteTimer.Stop()
			print("sending order complete message")

			for i := range E.Input.LastClearedButtons {
				orderMessage := network.Request{ButtonEvent: E.Input.LastClearedButtons[i],
					ID:        ID,
					Orders:    E.Output.LocalOrders,
					RequestID: NumRequests}

				go network.SendRequestUpdate(OrderCompletedTX, orderMessage, NumRequests)
				NumRequests++
				E.Input.LastClearedButtons = RemoveClearedOrder(lastClearedButtons, E.Input.LastClearedButtons[i])

			}

		case <-E.OrderCompleteTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, possible engine failure or faulty sensor")
		case <-E.ObstructionTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, door obstruction")
		case <-statusTicker.C:

			nodeStatusTX <- network.SingleElevatorStatus{ID: ID, PrevFloor: E.Input.PrevFloor, MotorDirection: E.Output.MotorDirection, Orders: E.Output.LocalOrders}

		}

	}

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
