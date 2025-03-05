package main

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"Sanntid/pba"
	"flag"

	//"Network-go/network/localip"
	//"Network-go/network/peers"
	//"fmt"
	"time"
)

func startDoorTimer(doorTimeout chan<- bool) {

	time.AfterFunc(3*time.Second, func() {
		doorTimeout <- true
	})
}

var StartingAsPrimary = flag.Bool("StartingAsPrimary", false, "Start as primary")

func main() {
	flag.Parse()

	var ID = time.Now().Format("20060102150405")
	if *StartingAsPrimary {
		fsm.PrimaryID = ID
	}

	peerTX := make(chan bool)

	go peers.Transmitter(12055, ID, peerTX)

	numFloors := 4
	elevio.Init("localhost:15657", numFloors)

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
	TXFloorReached := make(chan int)


	time.Sleep(1 * time.Second)

	go elevio.PollButtons(newOrder)
	go elevio.PollFloorSensor(floorReached)
	go bcast.Transmitter(13057, TXOrderCh)
	go bcast.Transmitter(13055, TXFloorReached)
	go bcast.Receiver(13056, RXOrderCh)

	for {
		select {
		case a := <-newOrder:
			OrderToPrimary := fsm.Order{
				ButtonEvent: a,
				ID:          ID,
				TargetID:   fsm.PrimaryID,
				Orders: storedInput.PressedButtons,
			}
			TXOrderCh <- OrderToPrimary
		case a := <-RXOrderCh:
			if a.TargetID != ID {
				continue
			}
			// While buttonlight off, spam order recieved
			if fsm.QueueEmpty(storedInput.PressedButtons) {
				doorTimeout <- true
			}
			storedInput.PressedButtons = a.Orders
			
		case a := <-floorReached:
			elevio.SetFloorIndicator(a)
			prevDirection := storedOutput.MotorDirection
			decision := fsm.HandleFloorReached(a, storedInput, storedOutput)
			state = decision.NextState
			storedInput.PrevFloor = a
			for i := 0; i < fsm.NButtons; i++ {
				for j := 0; j < fsm.NFloors; j++ {
					if decision.ElevatorOutput.ButtonLights[j][i] {
						elevio.SetButtonLamp(elevio.ButtonType(i), j, true)
					} else {
						elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
					}
				}
			}
			if state == fsm.DoorOpen {
				go startDoorTimer(doorTimeout)
				elevio.SetDoorOpenLamp(true)
			}
			storedOutput.MotorDirection = prevDirection
			storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights
			storedOutput.ButtonLights = decision.ElevatorOutput.ButtonLights
			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)

			// while buttonlight on, spam floor reached
			TXFloorReached <- a
		case <-doorTimeout:
			storedInput.PrevFloor = elevio.GetFloor()
			elevio.SetDoorOpenLamp(false)
			decision := fsm.HandleDoorTimeout(storedInput, storedOutput)
			state = decision.NextState
			if state == fsm.DoorOpen {
				go startDoorTimer(doorTimeout)
				elevio.SetDoorOpenLamp(true)
			}
			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
			storedOutput.MotorDirection = decision.ElevatorOutput.MotorDirection
		}
	}

}

