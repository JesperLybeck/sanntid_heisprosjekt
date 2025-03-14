package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"Sanntid/pba"
	"fmt"
	"os"
	"time"
)

func startDoorTimer(doorTimeout chan<- bool) {

	time.AfterFunc(3*time.Second, func() {
		doorTimeout <- true
	})
}

// ----------------------------ENVIRONMENT VARIABLES-----------------------------
var ID string
var startingAsPrimaryEnv string
var startingAsPrimary bool
var elevioPortNumber string

func main() {
	//-----------------------------CHANNELS-----------------------------
	peerTX := make(chan bool)
	nodeStatusTX := make(chan fsm.SingleElevatorStatus)
	buttonPressCh := make(chan elevio.ButtonEvent)
	floorReachedCh := make(chan int)
	doorTimeoutCh := make(chan bool)
	RequestTX := make(chan fsm.Order)
	RXOrderFromPrimCh := make(chan fsm.Order)
	TXFloorReached := make(chan fsm.Order)
	RXLightUpdate := make(chan fsm.LightUpdate)

	//vi skiller mellom intern komunikasjon og kommunikasjon med primary.
	//channels for internal communication is denoted with "eventCh" channels for external comms are named "EventTX or RX"

	//-----------------------------TIMERS-----------------------------
	statusTicker := time.NewTicker(2 * time.Second)

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
		elevioPortNumber = "localhost:15657" // Default value if the environment variable is not set
	}
	//-----------------------------STATE MACHINE VARIABLES-----------------------------
	var elevator fsm.Elevator

	//-----------------------------INITIALIZING ELEVATOR-----------------------------
	elevio.Init(elevioPortNumber, fsm.NFloors) //when starting, the elevator goes up until it reaches a floor.
	for elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Up)
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
	elevator.Output.Door = false
	elevator.Input.PrevFloor = elevio.GetFloor()
	elevator.Output.MotorDirection = elevio.MD_Stop
	elevio.SetMotorDirection(elevator.Output.MotorDirection)

	//-----------------------------GO ROUTINES-----------------------------
	go pba.Primary(ID) //starting go routines for primary and backup.
	go pba.Backup(ID)
	go elevio.PollButtons(buttonPressCh) //starting go routines for polling HW
	go elevio.PollFloorSensor(floorReachedCh)
	go bcast.Transmitter(13057, RequestTX) //starting go routines for network communication with other primary.
	go bcast.Receiver(13056, RXOrderFromPrimCh)
	go bcast.Transmitter(13058, TXFloorReached)
	go bcast.Transmitter(13059, nodeStatusTX)
	go bcast.Receiver(13060, RXLightUpdate)
	go peers.Transmitter(12055, ID, peerTX)

	//-----------------------------MAIN LOOP-----------------------------
	for {
		select {

		case lights := <-RXLightUpdate: //when light update is received from primary, the node updates its own lights with the newest information.
			if lights.ID == ID {
				for i := 0; i < fsm.NButtons; i++ {
					for j := 0; j < fsm.NFloors; j++ {
						if lights.LightArray[j][i] {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, true) // vi kan vurdere om denne faktisk kan pakkes inn i en funksjon da vi gjør dette flere steder  koden.
						} else {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
						}
					}
				}
			}
		case btnEvent := <-buttonPressCh: //case for å håndtere knappe trykk. Sender ordre til prim.			// Hvis heisen er i etasje n og får knappetrykk i n trenger man ikke å sende ordre til primary

			switch btnEvent.Button {

			case elevio.BT_Cab: //Jeg mener vi ikk trenger to caser. Dersom noden er på nett, skal den sende til prim.
				OrderToPrimary := fsm.Order{ //dersom noden ikke er på nett, vil den tro den er prim, og sende til seg selv. Begge tilfeller bør fungere!
					ButtonEvent: btnEvent,
					ID:          ID,
					TargetID:    fsm.PrimaryID,
					Orders:      elevator.Input.LocalRequests,
				}
				RequestTX <- OrderToPrimary
				// Hva gjør vi med cab calls når internett er nede. TODO: Implementer ONLINE/OFFLINE//ikke nødvendig. Se ovenfor.
			default:

				OrderToPrimary := fsm.Order{
					ButtonEvent: btnEvent,
					ID:          ID,
					TargetID:    fsm.PrimaryID,
					Orders:      elevator.Input.LocalRequests,
				}
				RequestTX <- OrderToPrimary
			}

		case a := <-RXOrderFromPrimCh: // I denne casen mottar noden en ordre fra primary.

			if a.TargetID != ID { // hvis ordren er til en annen heis, ignorer.
				continue
			}

			if a.ButtonEvent.Floor == elevio.GetFloor() && a.TargetID == ID { //hvis vi får en ordre på etasjen vi er på. Åpne dør.
				//ideelt sett får vi denne logikken bakt inn i en rent logisk "handle new order", og at vi slipper å håndtere spesialtilfellet her.
				elevator.State = fsm.DoorOpen
				go startDoorTimer(doorTimeoutCh)
				elevio.SetDoorOpenLamp(true)
				continue

			}
			//samme gjelder her. Kan vi unngå å håndtere spesiltilfeller her?
			//mener bestemt det bør være mulig å kunne formulere en handle new order som er dekkende for alle gyldige tilfeller.
			if fsm.QueueEmpty(elevator.Input.LocalRequests) { //dette vil også fungere når det kommer til å håndtere cab ordre som lastes inn.
				print("queue empty")
				elevator.Input.LocalRequests = a.Orders
				decision := fsm.HandleDoorTimeout(elevator.Input, elevator.Output)
				elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
				elevator.Output.MotorDirection = decision.ElevatorOutput.MotorDirection
			} else {
				elevator.Input.LocalRequests = a.Orders
			}

			elevio.SetMotorDirection(elevator.Output.MotorDirection)

			//vi mangler logikk for å håndtere ny ordre når køen ikke er tom!

		case a := <-floorReachedCh: //D

			elevio.SetFloorIndicator(a)
			prevDirection := elevator.Output.MotorDirection
			decision := fsm.HandleFloorReached(a, elevator.Input, elevator.Output)
			elevator.Input.PrevFloor = elevio.GetFloor()
			elevator.State = decision.NextState
			elevator.Input.PrevFloor = a

			fmt.Print("før", elevator.Input.LocalRequests)
			if elevator.State == fsm.DoorOpen {
				go startDoorTimer(doorTimeoutCh)
				elevio.SetDoorOpenLamp(true)
			}
			elevator.Input.LocalRequests = decision.ElevatorOutput.LocalOrders //funksjonen må endres sånn at den returnerer pressed buttons istedet.
			elevator.Output.MotorDirection = prevDirection
			fmt.Print("etter", elevator.Input.LocalRequests)

			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection) //vi må i stedet styre lys kun fra primary.

			// while buttonlight on, spam floor reached. Umulig, ingen funksjon som leser lysene

		case <-doorTimeoutCh:
			elevio.SetDoorOpenLamp(false)
			decision := fsm.HandleDoorTimeout(elevator.Input, elevator.Output)
			elevator.State = decision.NextState
			if elevator.State == fsm.DoorOpen {
				go startDoorTimer(doorTimeoutCh)
				elevio.SetDoorOpenLamp(true)
			}

			ArrivalMessage := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: elevio.GetFloor()}, ID: ID, TargetID: fsm.PrimaryID, Orders: elevator.Input.LocalRequests}
			for range 5 {
				TXFloorReached <- ArrivalMessage
			}
			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
			elevator.Output.MotorDirection = decision.ElevatorOutput.MotorDirection

		case <-statusTicker.C:

			nodeStatusTX <- fsm.SingleElevatorStatus{ID: ID, PrevFloor: elevio.GetFloor(), MotorDirection: elevator.Output.MotorDirection}
		}
	}

}
