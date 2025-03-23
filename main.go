package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"Sanntid/pba"
	"os"
	"time"
)

// ------------------------------Utility functions--------------------------------

func setHardwareEffects(E fsm.Elevator) {
	elevio.SetMotorDirection(E.Output.MotorDirection)
	elevio.SetDoorOpenLamp(E.Output.Door)
	elevio.SetFloorIndicator(E.Input.PrevFloor)

}

// ----------------------------ENVIRONMENT VARIABLES-----------------------------
var ID string
var startingAsPrimaryEnv string
var startingAsPrimary bool
var elevioPortNumber string

func main() {
	//-----------------------------CHANNELS-----------------------------
	peerTX := make(chan bool)
	nodeStatusTX := make(chan fsm.SingleElevatorStatus) //strictly having both should be unnecessary.
	RequestToPrimTX := make(chan fsm.Request)
	OrderFromPrimRX := make(chan fsm.Order)
	OrderCompletedTX := make(chan fsm.Request)
	LightUpdateFromPrimRX := make(chan fsm.LightUpdate)
	peersRX := make(chan peers.PeerUpdate)
	primStatusRX := make(chan fsm.Status)

	buttonPressCh := make(chan elevio.ButtonEvent)
	floorReachedCh := make(chan int)

	//doorTimeoutCh := make(chan bool)

	//vi skiller mellom intern komunikasjon og kommunikasjon med primary.
	//channels for internal communication is denoted with "eventCh" channels for external comms are named "EventTX or RX"

	//-----------------------------TIMERS-----------------------------
	statusTicker := time.NewTicker(50 * time.Millisecond)

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

	if startingAsPrimary {
		fsm.PrimaryID = ID
	}

	elevioPortNumber = os.Getenv("PORT") // Read the environment variable
	if elevioPortNumber == "" {
		elevioPortNumber = "15657" // Default value if the environment variable is not set
	}
	//-----------------------------STATE MACHINE VARIABLES-----------------------------
	var elevator fsm.Elevator
	var prevLightMatrix [fsm.NFloors][fsm.NButtons]bool
	var NumRequests int = 1
	var lastOrderID int = 0

	//-----------------------------INITIALIZING ELEVATOR-----------------------------
	elevio.Init("localhost:"+elevioPortNumber, fsm.NFloors) //when starting, the elevator goes up until it reaches a floor.

	for {
		elevio.SetMotorDirection(elevio.MD_Up)
		if elevio.GetFloor() != -1 {
			elevio.SetMotorDirection(elevio.MD_Stop)
			break
		}
		time.Sleep(fsm.OrderTimeout * time.Millisecond) // Add small delay between polls
	}
	for j := 0; j < 4; j++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, j, false) //skrur av alle lys ved initsialisering. Nødvendig???
		elevio.SetButtonLamp(elevio.BT_HallDown, j, false)
		elevio.SetButtonLamp(elevio.BT_Cab, j, false)
	}
	//elevator state machine variables are initialized.
	elevator.State = fsm.Idle                                        //after initializing the elevator, it goes to the idle state.
	elevator.Input.LocalRequests = [fsm.NFloors][fsm.NButtons]bool{} // not strictly necessary, but...
	elevator.Output.LocalOrders = [fsm.NFloors][fsm.NButtons]bool{}
	elevator.Input.PrevFloor = elevio.GetFloor()
	elevator.DoorTimer = time.NewTimer(3 * time.Second)
	elevator.OrderCompleteTimer = time.NewTimer(fsm.OrderTimeout * time.Second)
	elevator.ObstructionTimer = time.NewTimer(7 * time.Second)

	elevator.ObstructionTimer.Stop()
	elevator.DoorTimer.Stop()
	elevator.OrderCompleteTimer.Stop()
	//-----------------------------GO ROUTINES-----------------------------
	go pba.Primary(ID) //starting go routines for primary and backup.
	go pba.Backup(ID)

	go elevio.PollButtons(buttonPressCh) //starting go routines for polling HW
	go elevio.PollFloorSensor(floorReachedCh)
	go bcast.Transmitter(13057, RequestToPrimTX) //starting go routines for network communication with other primary.
	go bcast.Receiver(13056, OrderFromPrimRX)
	go bcast.Transmitter(13058, OrderCompletedTX)
	go bcast.Transmitter(13059, nodeStatusTX)
	go bcast.Receiver(13060, LightUpdateFromPrimRX)
	go peers.Transmitter(12055, ID, peerTX)
	go peers.Receiver(12055, peersRX)
	go bcast.Receiver(13055, primStatusRX)

	//-----------------------------MAIN LOOP-----------------------------
	for {
		select {
		case primStatus := <-primStatusRX:
			//Hva om vi gjør en del av dette her, heller enn i pba?
			fsm.IpToIndexMap = primStatus.Map
			fsm.LatestPeerList = primStatus.Peerlist

		case p := <-peersRX: //vi klarer oss vell uten denne, med casen over?
			// To register if alone on network and enter offline mode
			if len(p.Peers) == 0 {
				fsm.AloneOnNetwork = true
			}
		case lights := <-LightUpdateFromPrimRX:

			//when light update is received from primary, the node updates its own lights with the newest information.
			if (fsm.LightsDifferent(prevLightMatrix, lights.LightArray)) && lights.ID == ID {

				for i := range fsm.NButtons {
					for j := range fsm.NFloors {
						elevio.SetButtonLamp(elevio.ButtonType(i), j, lights.LightArray[j][i]) // vi kan vurdere om denne faktisk kan pakkes inn i en funksjon da vi gjør dette flere steder  koden.

					}

					prevLightMatrix = lights.LightArray
				}
			}

		case btnEvent := <-buttonPressCh: //case for å håndtere knappe trykk. Sender ordre til prim.			// Hvis heisen er i etasje n og får knappetrykk i n trenger man ikke å sende ordre til primary
			elevator.Input.LocalRequests[btnEvent.Floor][btnEvent.Button] = true
			requestToPrimary := fsm.Request{
				ButtonEvent: btnEvent,
				ID:          ID,
				TargetID:    fsm.PrimaryID,
				Orders:      elevator.Input.LocalRequests,
				RequestID:   NumRequests,
			}
			// ISSUE! when the order is delegated to a different node, we cant ack on
			go fsm.SendRequestUpdate(RequestToPrimTX, primStatusRX, requestToPrimary, NumRequests)
			NumRequests++

			if fsm.AloneOnNetwork && btnEvent.Button == elevio.BT_Cab {
				offlineOrder := fsm.Order{ButtonEvent: btnEvent, ResponisbleElevator: ID}
				elevator = fsm.HandleNewOrder(offlineOrder, elevator) //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.
				elevio.SetButtonLamp(elevio.BT_Cab, btnEvent.Floor, true)
				setHardwareEffects(elevator)
			}

			//vi diskuterte om vi trengte å ha egen case for cab. Vi kom frem til at det ikke trengs fordi:
			//i: Hvis noden har kræsjet, tar vi ikke ordre.
			//ii: Hvis noden er uten nett, setter den seg selv til primary, den vil dermed ta ordre fra cab selv.

		case order := <-OrderFromPrimRX: // I denne casen mottar noden en ordre fra primary.

			if order.ResponisbleElevator != ID || order.ResponisbleElevator == ID && lastOrderID == order.OrderID { // hvis ordren er til en annen heis, ignorer.
				//denne kunne også strengt tatt gått inn i handle new Order functionen.
				print("anti spam order received from prim")
				continue
			}
			print("handling new order")
			//problem om heisen allerede er i etasjen ordren er i.
			//Da vil primary ikke få ack, fordi handle new order legger ikke til ordren i localOrders.
			//programmet terminerer ikke.
			elevator = fsm.HandleNewOrder(order, elevator)
			lastOrderID = order.OrderID //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.

			setHardwareEffects(elevator)

		case a := <-floorReachedCh:

			elevator = fsm.HandleFloorReached(a, elevator)

			setHardwareEffects(elevator)

			if fsm.AloneOnNetwork {
				elevio.SetButtonLamp(elevio.BT_Cab, a, false)
			}

		case <-elevator.DoorTimer.C:

			elevator.DoorObstructed = elevio.GetObstruction()

			elevator = fsm.HandleDoorTimeout(elevator)

			setHardwareEffects(elevator)

			orderMessage := fsm.Request{ButtonEvent: elevio.ButtonEvent{Floor: elevio.GetFloor()},
				ID:        ID,
				TargetID:  fsm.PrimaryID,
				Orders:    elevator.Output.LocalOrders,
				RequestID: NumRequests}

			elevator.OrderCompleteTimer.Stop()
			print("sending order complete message")
			go fsm.SendRequestUpdate(OrderCompletedTX, primStatusRX, orderMessage, NumRequests)
			NumRequests++

		case <-elevator.OrderCompleteTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, possible engine failure or faulty sensor")
		case <-elevator.ObstructionTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, door obstruction")
		case <-statusTicker.C:

			nodeStatusTX <- fsm.SingleElevatorStatus{ID: ID, PrevFloor: elevio.GetFloor(), MotorDirection: elevator.Output.MotorDirection, Orders: elevator.Output.LocalOrders}

		}

	}

}
