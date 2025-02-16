package main

import (
	"Sanntid/elevio"
	"Sanntid/fsm"
	"fmt"
	"time"
)

func testHandleFloorReached() {
	event := 0
	storedInput := fsm.ElevatorInput{
		PressedButtons: [4][3]bool{
			{false, false, false},
			{false, false, false},
			{true, false, false}, // Simulerer en knappetrykk på etasje 2
			{false, false, false},
		},
		PrevFloor: 1,
	}
	storedOutput := fsm.ElevatorOutput{
		MotorDirection: elevio.MD_Down,
		Door:           false,
		ButtonLights: [4][3]bool{
			{false, false, false},
			{false, false, false},
			{false, false, false}, // Simulerer en knappetrykk på etasje 2
			{true, false, false},
		},
	}

	decision := fsm.HandleFloorReached(event, storedInput, storedOutput)

	fmt.Println("Motor Direction:", decision.ElevatorOutput.MotorDirection)

}

func startDoorTimer(doorTimeout chan<- bool) {

	time.AfterFunc(3*time.Second, func() {
		doorTimeout <- true
	})
}

func main() {
	testHandleFloorReached()
	numFloors := 4
	elevio.Init("localhost:15657", numFloors)

	println("Initializing elevator")
	for elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Up)
	}

	for j := 0; j < 4; j++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, j, false)
		elevio.SetButtonLamp(elevio.BT_HallDown, j, false)
		elevio.SetButtonLamp(elevio.BT_Cab, j, false)
	}

	elevio.SetMotorDirection(elevio.MD_Stop)
	println("start floor reached")
	println(elevio.GetFloor())

	var state fsm.ElevatorState = fsm.Idle
	var storedInput fsm.ElevatorInput
	storedInput.PrevFloor = elevio.GetFloor()

	var storedOutput fsm.ElevatorOutput
	storedOutput.MotorDirection = elevio.MD_Stop

	newOrder := make(chan elevio.ButtonEvent)
	floorReached := make(chan int)
	doorTimeout := make(chan bool)
	time.Sleep(1 * time.Second)

	go elevio.PollButtons(newOrder)
	go elevio.PollFloorSensor(floorReached)

	for {
		select {
		case a := <-newOrder:

			fmt.Println("3?", state)
			elevio.SetButtonLamp(a.Button, a.Floor, true)
			storedInput.PressedButtons[a.Floor][a.Button] = true
			storedOutput.ButtonLights = storedInput.PressedButtons
			switch state {
			case fsm.Idle:

				decision := fsm.HandleNewOrder(state, a, storedInput, storedOutput)
				state = decision.NextState
				storedOutput = decision.ElevatorOutput
				elevio.SetMotorDirection(storedOutput.MotorDirection)
				storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights

				fmt.Println(decision.ElevatorOutput.ButtonLights, "buttonlights from descision")

				for i := 0; i < 3; i++ {
					for j := 0; j < 4; j++ {
						if decision.ElevatorOutput.ButtonLights[j][i] {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, true)
						} else {
							elevio.SetButtonLamp(elevio.ButtonType(i), j, false)
						}
					}
				}
				if a.Floor == elevio.GetFloor() {

					go startDoorTimer(doorTimeout)
					elevio.SetDoorOpenLamp(true)

				}

			case fsm.MovingBetweenFloors:
				decision := fsm.HandleNewOrder(state, a, storedInput, storedOutput)
				state = decision.NextState
				storedOutput = decision.ElevatorOutput
				storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights
				fmt.Println(storedInput.PressedButtons, "pressed buttons after new order")
				fmt.Println(storedOutput.ButtonLights, "store buttonlights after NEW order")

			case fsm.DoorOpen:
				decision := fsm.HandleNewOrder(state, a, storedInput, storedOutput)
				state = decision.NextState
				storedOutput = decision.ElevatorOutput
				storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights

				//case fsm.MovingPassingFloor:

			}
		case a := <-floorReached:
			fmt.Println("Floor reached")

			elevio.SetFloorIndicator(a)
			prevDirection := storedOutput.MotorDirection
			println("prevDirection", prevDirection)
			decision := fsm.HandleFloorReached(a, storedInput, storedOutput)
			state = decision.NextState
			storedInput.PrevFloor = a

			for i := 0; i < 3; i++ {
				for j := 0; j < 4; j++ {
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

		case <-doorTimeout:

			elevio.SetDoorOpenLamp(false)
			state = fsm.Idle
			decision := fsm.HandleDoorTimeout(storedInput, storedOutput)
			state = decision.NextState
			elevio.SetMotorDirection(decision.ElevatorOutput.MotorDirection)
			storedOutput.MotorDirection = decision.ElevatorOutput.MotorDirection
			elevio.SetDoorOpenLamp(false)
			println("door sequece done")

		}
	}

}
