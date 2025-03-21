package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Sanntid/config"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"Sanntid/pba"
	"fmt"
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
	// Main channels
	peerTX := make(chan bool)
	nodeStatusTX := make(chan config.SingleElevatorStatus) //strictly having both should be unnecessary.
	RequestToPrimTX := make(chan config.Order)
	OrderFromPrimRX := make(chan config.Order)
	OrderCompletedTX := make(chan config.Order)
	LightUpdateFromPrimRX := make(chan config.LightUpdate)
	buttonPressCh := make(chan elevio.ButtonEvent)
	floorReachedCh := make(chan int)

	// Primary channels
	channels := pba.PrimaryChannels{
		OrderTX:        make(chan config.Order),
		OrderRX:        make(chan config.Order),
		RXFloorReached: make(chan config.Order),
		StatusTX:       make(chan config.Status),
		NodeStatusRX:   make(chan config.SingleElevatorStatus),
		TXLightUpdates: make(chan config.LightUpdate),
	}

	// Backup and listener channels
	primaryStatusRX := make(chan config.Status)

	// Shared channels
	peersRX := make(chan peers.PeerUpdate)

	

	//vi skiller mellom intern komunikasjon og kommunikasjon med primary.
	//channels for internal communication is denoted with "eventCh" channels for external comms are named "EventTX or RX"

	//-----------------------------TIMERS-----------------------------
	statusTicker := time.NewTicker(200 * time.Millisecond)

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
		config.PrimaryID = ID
	}

	elevioPortNumber = os.Getenv("PORT") // Read the environment variable
	if elevioPortNumber == "" {
		elevioPortNumber = "15657" // Default value if the environment variable is not set
	}
	//-----------------------------STATE MACHINE VARIABLES-----------------------------
	var elevator fsm.Elevator

	//-----------------------------INITIALIZING ELEVATOR-----------------------------
	elevio.Init("localhost:"+elevioPortNumber, config.NFloors) //when starting, the elevator goes up until it reaches a floor.

	for {
		elevio.SetMotorDirection(elevio.MD_Up)
		if elevio.GetFloor() != -1 {
			elevio.SetMotorDirection(elevio.MD_Stop)
			break
		}
		time.Sleep(config.OrderTimeout * time.Millisecond) // Add small delay between polls
	}
	for j := 0; j < 4; j++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, j, false) //skrur av alle lys ved initsialisering. Nødvendig???
		elevio.SetButtonLamp(elevio.BT_HallDown, j, false)
		elevio.SetButtonLamp(elevio.BT_Cab, j, false)
	}
	//elevator state machine variables are initialized.
	elevator.State = fsm.Idle                                        //after initializing the elevator, it goes to the idle state.
	elevator.Input.LocalRequests = [config.NFloors][config.NButtons]bool{} // not strictly necessary, but...
	elevator.Output.LocalOrders = [config.NFloors][config.NButtons]bool{}
	elevator.Output.Door = false
	elevator.Input.PrevFloor = elevio.GetFloor()
	elevator.DoorTimer = time.NewTimer(3 * time.Second)
	elevator.OrderCompleteTimer = time.NewTimer(config.OrderTimeout * time.Second)
	elevator.DoorTimer.Stop()
	elevator.OrderCompleteTimer.Stop()
	//-----------------------------GO ROUTINES-----------------------------
	go pba.Primary(ID, channels, peersRX) //starting go routines for primary, backup and active listener
	go pba.Backup(ID, primaryStatusRX)
	go pba.StatusReciever(ID, primaryStatusRX)

	go elevio.PollButtons(buttonPressCh) //starting go routines for polling HW
	go elevio.PollFloorSensor(floorReachedCh)
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
		case p := <-peersRX:
			// To register if alone on network and enter offline mode
			if len(p.Peers) == 0 {
				config.AloneOnNetwork = true
			}
		case lights := <-LightUpdateFromPrimRX:
			//when light update is received from primary, the node updates its own lights with the newest information.
			if lights.ID == ID {
				for i := range config.NButtons {
					for j := range config.NFloors {
						elevio.SetButtonLamp(elevio.ButtonType(i), j, lights.LightArray[j][i]) // vi kan vurdere om denne faktisk kan pakkes inn i en funksjon da vi gjør dette flere steder  koden.

					}
				}
			}

		case btnEvent := <-buttonPressCh: //case for å håndtere knappe trykk. Sender ordre til prim.			// Hvis heisen er i etasje n og får knappetrykk i n trenger man ikke å sende ordre til primary

			requestToPrimary := config.Order{
				ButtonEvent: btnEvent,
				ID:          ID,
				TargetID:    config.PrimaryID,
				Orders:      elevator.Input.LocalRequests,
			}


			RequestToPrimTX <- requestToPrimary
			fmt.Println("Sent order to primary: ", requestToPrimary)
			

			if config.AloneOnNetwork && btnEvent.Button == elevio.BT_Cab {
				
				elevator = fsm.HandleNewOrder(requestToPrimary, elevator) //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.
				elevio.SetButtonLamp(elevio.BT_Cab, btnEvent.Floor, true)
				setHardwareEffects(elevator)
			}

			//vi diskuterte om vi trengte å ha egen case for cab. Vi kom frem til at det ikke trengs fordi:
			//i: Hvis noden har kræsjet, tar vi ikke ordre.
			//ii: Hvis noden er uten nett, setter den seg selv til primary, den vil dermed ta ordre fra cab selv.

		case order := <-OrderFromPrimRX: // I denne casen mottar noden en ordre fra primary.

			if order.TargetID != ID { // hvis ordren er til en annen heis, ignorer.
				//denne kunne også strengt tatt gått inn i handle new Order functionen.
				continue
			}
			print("Received order from primary: ")

			elevator = fsm.HandleNewOrder(order, elevator) //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.

			setHardwareEffects(elevator)

		case a := <-floorReachedCh:
	
			elevator = fsm.HandleFloorReached(a, elevator)

			setHardwareEffects(elevator)

			if config.AloneOnNetwork {
				elevio.SetButtonLamp(elevio.BT_Cab, a, false)
			}


		case <-elevator.DoorTimer.C:
			elevator.DoorObstructed = elevio.GetObstruction()
			elevator = fsm.HandleDoorTimeout(elevator)

			setHardwareEffects(elevator)

			orderMessage := config.Order{ButtonEvent: elevio.ButtonEvent{Floor: elevio.GetFloor()}, ID: ID, TargetID: config.PrimaryID, Orders: elevator.Output.LocalOrders}
			elevator.OrderCompleteTimer.Stop()
			OrderCompletedTX <- orderMessage
		case <-elevator.OrderCompleteTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, possible engine failure or faulty sensor")
		case <-statusTicker.C:

			nodeStatusTX <- config.SingleElevatorStatus{ID: ID, PrevFloor: elevio.GetFloor(), MotorDirection: elevator.Output.MotorDirection}

		}

	}

}
