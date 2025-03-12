package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"Sanntid/pba"
	"flag"

	//"Network-go/network/peers"

	"os"
	"time"
)

func startDoorTimer(doorTimeout chan<- bool) {

	time.AfterFunc(3*time.Second, func() {
		doorTimeout <- true
	})
}
func requestOrderAckTimer(requestAckTimeout chan<- bool) *time.Timer {
	return time.AfterFunc(3*time.Second, func() {
		requestAckTimeout <- true
	})
}

var StartingAsPrimary = flag.Bool("StartingAsPrimary", false, "Start as primary")

func main() {
	statusTicker := time.NewTicker(2 * time.Second)
	var p_requestOrderAckTimer *time.Timer

	flag.Parse()
	var ID string
	ID = os.Getenv("ID")
	if ID == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			localIP = "DISCONNECTED"
		}
		ID = localIP
	}

	println("myID", ID)
	peerTX := make(chan bool)
	nodeStatusTX := make(chan fsm.SingleElevatorStatus)
	requestAckTimeout := make(chan bool)
	//AliveTicker := time.NewTicker(2 * time.Second)

	go peers.Transmitter(12055, ID, peerTX)
	var StartingAsPrimary bool
	StartingAsPrimaryEnv := os.Getenv("STARTASPRIM")
	if StartingAsPrimaryEnv == "true" {
		StartingAsPrimary = true
		println("Starting as primary")
	} else {
		StartingAsPrimary = false
		println("Not starting as primary")
	}

	if StartingAsPrimary {
		fsm.PrimaryID = ID
	}

	elevioPortNumber := os.Getenv("PORT") // Read the environment variable
	if elevioPortNumber == "" {
		elevioPortNumber = "localhost:15657" // Default value if the environment variable is not set
	}
	println("Port number: ", elevioPortNumber)
	elevio.Init(elevioPortNumber, fsm.NFloors)

	for elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Up)
	}

	go pba.Primary(ID)
	go pba.Backup(ID)

	for j := 0; j < 4; j++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, j, false)
		elevio.SetButtonLamp(elevio.BT_HallDown, j, false)
		elevio.SetButtonLamp(elevio.BT_Cab, j, false)
	}

	elevio.SetMotorDirection(elevio.MD_Stop)

	var state fsm.ElevatorState = fsm.Idle
	var storedInput fsm.ElevatorInput
	storedInput.PrevFloor = elevio.GetFloor()

	var storedOutput fsm.ElevatorOutput
	storedOutput.MotorDirection = elevio.MD_Stop

	newOrder := make(chan elevio.ButtonEvent)
	floorReached := make(chan int)
	doorTimeout := make(chan bool)
	TXOrderCh := make(chan fsm.Order)
	RXOrderCh := make(chan fsm.Order)
	TXFloorReached := make(chan fsm.Order)

	time.Sleep(1 * time.Second)

	go elevio.PollButtons(newOrder)
	go elevio.PollFloorSensor(floorReached)
	go bcast.Transmitter(13057, TXOrderCh)
	go bcast.Receiver(13056, RXOrderCh)
	go bcast.Transmitter(13058, TXFloorReached)
	go bcast.Transmitter(13059, nodeStatusTX)

	for {
		select {

		case a := <-newOrder:

			// Hvis heisen er i etasje n og får knappetrykk i n trenger man ikke å sende ordre til primary
			// EVt bare cleare i retninga heisen går, ikke i motsatt retning
			//Jo send ordre til primary, assign elevator skal da velge heisen som allerede er i etasjen.

			switch a.Button {
			case elevio.BT_Cab:
				// Hva gjør vi med cab calls når internett er nede. TODO: Implementer ONLINE/OFFLINE
			default:

				OrderToPrimary := fsm.Order{
					ButtonEvent: a,
					ID:          ID,
					TargetID:    fsm.PrimaryID,
					Orders:      storedInput.PressedButtons,
				}
				TXOrderCh <- OrderToPrimary
			}
		case a := <-RXOrderCh:
			queueWasEmpty := fsm.QueueEmpty(storedInput.PressedButtons)
			//uavhengig av hvilken nodeordren er til, skal vi oppdatere lysene for hall.
			// En ordre som er kommet hit fra primary er skal være lagret av backup. Knappelys kan dermed skrus på her, så lengde det ikke er cab call.
			if a.ID[0:3] == fsm.AckPrefix {

				if a.ID[3:] == ID { //heisen som har utført ordren, er ansvarlig for å stoppe timeren.
					p_requestOrderAckTimer.Stop() //stopper watchdog timeren på ack.
					println("stopping ack timer")

				}
				//oppdaterer
				//dette er en kvittering på at primary og backup registrert en ordre som utført.
				a.ID = a.ID[3:]
			}
			//oppdaterer lysene på hall. Dette gjøres for alle nodene.
			for floor := 0; floor < fsm.NFloors; floor++ {
				for button := 0; button < fsm.NButtons-1; button++ { // -1 to exclude cab calls
					storedInput.PressedButtons[floor][button] = a.Orders[floor][button]

				}
			}
			if a.TargetID == ID {
				for floor := 0; floor < fsm.NFloors; floor++ {
					storedInput.PressedButtons[floor][2] = a.Orders[floor][2] //hvis man er target for meld, oppdateres også cab lights
				}
			}

			for i := 0; i < fsm.NButtons-1; i++ {
				for j := 0; j < fsm.NFloors; j++ {
					if storedInput.PressedButtons[j][i] {
						elevio.SetButtonLamp(elevio.ButtonType(i), j, true)
					} else {
						elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
					}
				}

			}
			if a.TargetID != ID {
				continue //hvis ordren ikke er til denne noden, trenger vi ikke gjøre noe mer.

			}
			print("order for me: ", ID)
			//fmt.Print("Order recieved", ID, "floor", a.ButtonEvent.Floor, storedInput.PressedButtons)

			if queueWasEmpty {
				//storedInput.PressedButtons = a.Orders
				print("queue was empty")
				decision := fsm.HandleDoorTimeout(storedInput, storedOutput) //sketchy navngivning.
				print(decision.ElevatorOutput.MotorDirection)
				elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
				storedOutput.MotorDirection = decision.ElevatorOutput.MotorDirection
			}

			/*
				if a.ButtonEvent.Floor == elevio.GetFloor() { //vi er allerede her? trigg arrived at floor//Hvis vi ikke sender ordre på etasjen man befinner seg i til prim, trenger vi ikke denne.
					//floorReached <- a.ButtonEvent.Floor

				}
				if a.ButtonEvent.Button != elevio.BT_Cab {
					elevio.SetButtonLamp(a.ButtonEvent.Button, a.ButtonEvent.Floor, true)
				}
				if a.TargetID != ID {
					continue
				}
				if a.TargetID == ID {
					print("Order recieved", a.ID, "floor", a.ButtonEvent.Floor)
				}

				// While buttonlight off, spam order recieved. Umulig, ingen funksjon som leser lysene
				if fsm.QueueEmpty(storedInput.PressedButtons) {
					storedInput.PressedButtons = a.Orders
					decision := fsm.HandleDoorTimeout(storedInput, storedOutput)
					elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
					storedOutput.MotorDirection = decision.ElevatorOutput.MotorDirection
				}
				storedInput.PressedButtons = a.Orders
				for i := 0; i < fsm.NButtons; i++ {
					for j := 0; j < fsm.NFloors; j++ {
						if storedInput.PressedButtons[j][i] {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, true)
						} else {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
						}
					}
				}*/

		case a := <-floorReached:
			elevio.SetFloorIndicator(a)
			prevDirection := storedOutput.MotorDirection
			decision := fsm.HandleFloorReached(a, storedInput, storedOutput)
			storedInput.PrevFloor = elevio.GetFloor()
			state = decision.NextState
			storedInput.PrevFloor = a
			/*
				for i := 0; i < fsm.NButtons; i++ {
					for j := 0; j < fsm.NFloors; j++ {
						if decision.ElevatorOutput.ButtonLights[j][i] { //buttonlights kan ikke bestemmer lokalt, dette må styres av primary,

							elevio.SetButtonLamp(elevio.ButtonType(i), j, true) //vi må heller sende melding til primary at ordren er utført.
						} else {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
						}
					}
				}*/
			if state == fsm.DoorOpen { //er det ikke rart at vi kan nå en etasje og samtidig ha dør åpen?
				go startDoorTimer(doorTimeout)
				elevio.SetDoorOpenLamp(true)

			} //krever større endringer i fsm. Single elevator skal ikke lengre bestemme lysene. Mulig unntak må gjøres for cab.
			if fsm.OrderAtFloor(storedInput.PressedButtons, a) {
				ArrivalMessage := fsm.Order{ButtonEvent: elevio.ButtonEvent{}, ID: ID, TargetID: fsm.PrimaryID, Orders: storedInput.PressedButtons}
				for range 5 {
					TXFloorReached <- ArrivalMessage

				}
				print("sent order arrived message.")
				p_requestOrderAckTimer = requestOrderAckTimer(requestAckTimeout)
			}
			storedOutput.MotorDirection = prevDirection //hva?
			//storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights //dette kan ikke gjøres slik.
			//storedOutput.ButtonLights = decision.ElevatorOutput.ButtonLights  //dette skaper mismatch mellom de forskjellige nodene.

			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection) //vi må i stedet styre lys kun fra primary.

			//Her starter vi en timer, som skal vente på at prim har kvittert at ordren er utført. Hvis ingenting høres, antar vi noden er i offline modus.
		case <-doorTimeout:
			elevio.SetDoorOpenLamp(false)
			decision := fsm.HandleDoorTimeout(storedInput, storedOutput)
			state = decision.NextState
			if state == fsm.DoorOpen {
				go startDoorTimer(doorTimeout)
				elevio.SetDoorOpenLamp(true)
			}
			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
			storedOutput.MotorDirection = decision.ElevatorOutput.MotorDirection
		case <-statusTicker.C:

			nodeStatusTX <- fsm.SingleElevatorStatus{ID: ID, PrevFloor: elevio.GetFloor(), MotorDirection: storedOutput.MotorDirection}
		case <-requestAckTimeout:
			//Hvis vi ikke får ack fra backup, må vi anta vi er offline
			print("no ack from prim, assuming offline")
		}
	}

}
