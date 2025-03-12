package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"Sanntid/pba"
	"flag"
	"fmt"

	//"Network-go/network/peers"

	"os"
	"time"
)

func startDoorTimer(doorTimeout chan<- bool) {

	time.AfterFunc(3*time.Second, func() {
		doorTimeout <- true
	})
}

var StartingAsPrimary = flag.Bool("StartingAsPrimary", false, "Start as primary")

func main() {

	statusTicker := time.NewTicker(2 * time.Second)
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
	RXLightUpdate := make(chan fsm.LightUpdate)

	time.Sleep(1 * time.Second)

	go elevio.PollButtons(newOrder)
	go elevio.PollFloorSensor(floorReached)
	go bcast.Transmitter(13057, TXOrderCh)
	go bcast.Receiver(13056, RXOrderCh)
	go bcast.Transmitter(13058, TXFloorReached)
	go bcast.Transmitter(13059, nodeStatusTX)
	go bcast.Receiver(13060, RXLightUpdate)

	for {
		select {

		case lights := <-RXLightUpdate:
			if lights.ID == ID {
				for i := 0; i < fsm.NButtons; i++ {
					for j := 0; j < fsm.NFloors; j++ {
						if lights.LightArray[j][i] {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, true)
						} else {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
						}
					}
				}
			}
		case a := <-newOrder:

			// Hvis heisen er i etasje n og får knappetrykk i n trenger man ikke å sende ordre til primary
			// EVt bare cleare i retninga heisen går, ikke i motsatt retning
			//Jo send ordre til primary, assign elevator skal da velge heisen som allerede er i etasjen.

			switch a.Button {

			case elevio.BT_Cab:
				OrderToPrimary := fsm.Order{
					ButtonEvent: a,
					ID:          ID,
					TargetID:    fsm.PrimaryID,
					Orders:      storedInput.PressedButtons,
				}
				TXOrderCh <- OrderToPrimary
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

			if a.TargetID != ID { // hvis ordren er til en annen heis, ignorer.
				continue
			}
			//-----------------------------SINGLE ELEVATOR TING------------------
			print("Order recieved", a.TargetID)
			if a.ButtonEvent.Floor == elevio.GetFloor() && a.TargetID == ID {

				state = fsm.DoorOpen

				go startDoorTimer(doorTimeout)
				elevio.SetDoorOpenLamp(true) //letting primary know that the order is done.

				continue
				//hvis vi får en ordre når vi allerede er i etasjen, så skal vi åpne døra.
			}

			if fsm.QueueEmpty(storedInput.PressedButtons) {

				storedInput.PressedButtons = a.Orders
				decision := fsm.HandleDoorTimeout(storedInput, storedOutput)
				elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
				storedOutput.MotorDirection = decision.ElevatorOutput.MotorDirection
			}
			fmt.Print(a.ID, storedInput.PressedButtons)
			elevio.SetMotorDirection(storedOutput.MotorDirection)

			/*
				// En ordre som er kommet hit fra primary er skal være lagret av backup. Knappelys kan dermed skrus på her, så lengde det ikke er cab call.
				.ButtonEvent.Button != elevio.BT_Cab {
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
				}
			*/
		case a := <-floorReached:

			elevio.SetFloorIndicator(a)
			prevDirection := storedOutput.MotorDirection
			decision := fsm.HandleFloorReached(a, storedInput, storedOutput)
			storedInput.PrevFloor = elevio.GetFloor()
			state = decision.NextState
			storedInput.PrevFloor = a
			/*for i := 0; i < fsm.NButtons; i++ {
				for j := 0; j < fsm.NFloors; j++ {
					if decision.ElevatorOutput.ButtonLights[j][i] { //buttonlights kan ikke bestemmer lokalt, dette må styres av primary,

						elevio.SetButtonLamp(elevio.ButtonType(i), j, true) //vi må heller sende melding til primary at ordren er utført.
					} else {
						elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
					}
				}
			}*/
			if state == fsm.DoorOpen {
				go startDoorTimer(doorTimeout)
				elevio.SetDoorOpenLamp(true)
			}
			storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights //funksjonen må endres sånn at den returnerer pressed buttons istedet.
			storedOutput.MotorDirection = prevDirection
			//storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights //dette kan ikke gjøres slik.
			//storedOutput.ButtonLights = decision.ElevatorOutput.ButtonLights  //dette skaper mismatch mellom de forskjellige nodene.

			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection) //vi må i stedet styre lys kun fra primary.

			// while buttonlight on, spam floor reached. Umulig, ingen funksjon som leser lysene

		case <-doorTimeout:
			elevio.SetDoorOpenLamp(false)
			decision := fsm.HandleDoorTimeout(storedInput, storedOutput)
			state = decision.NextState
			if state == fsm.DoorOpen {
				go startDoorTimer(doorTimeout)
				elevio.SetDoorOpenLamp(true)
			}

			ArrivalMessage := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: elevio.GetFloor()}, ID: ID, TargetID: fsm.PrimaryID, Orders: storedInput.PressedButtons}
			for range 5 {
				TXFloorReached <- ArrivalMessage
			}
			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
			storedOutput.MotorDirection = decision.ElevatorOutput.MotorDirection

		case <-statusTicker.C:

			nodeStatusTX <- fsm.SingleElevatorStatus{ID: ID, PrevFloor: elevio.GetFloor(), MotorDirection: storedOutput.MotorDirection}
		}
	}

}
