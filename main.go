package main

import (
	"Sanntid/elevio"
	"Sanntid/fsm"
	"fmt"
	"time"
)

func startDoorTimer(doorTimeout chan<- bool) {
	time.Sleep(3 * time.Second)
	doorTimeout <- true
}

func main() {

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
			fmt.Println(storedInput.PressedButtons, "pressed buttons before new order")
			fmt.Println(storedOutput.ButtonLights, "store buttonlights before NEW order")
			fmt.Println(state)
			elevio.SetButtonLamp(a.Button, a.Floor, true)
			storedInput.PressedButtons[a.Floor][a.Button] = true
			switch state {
			case fsm.Idle:

				decision := fsm.HandleNewOrder(state, a, storedInput, storedOutput)
				state = decision.NextState
				storedOutput = decision.ElevatorOutput
				elevio.SetMotorDirection(storedOutput.MotorDirection)
				storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights
				fmt.Println(storedInput.PressedButtons, "pressed buttons before new order")
				fmt.Println(storedOutput.ButtonLights, "store buttonlights before NEW order")
				print("new order")

			case fsm.MovingBetweenFloors:
				decision := fsm.HandleNewOrder(state, a, storedInput, storedOutput)
				state = decision.NextState
				storedOutput = decision.ElevatorOutput
				storedInput.PressedButtons = decision.ElevatorOutput.ButtonLights
				fmt.Println(storedInput.PressedButtons, "pressed buttons after new order")
				fmt.Println(storedOutput.ButtonLights, "store buttonlights after NEW order")

				//case fsm.DoorOpen:

				//case fsm.MovingPassingFloor:

			}
		case a := <-floorReached:
			fmt.Println("Floor reached")

			elevio.SetFloorIndicator(a)
			prevDirection := storedOutput.MotorDirection
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
			elevio.SetDoorOpenLamp(false)
			println("door sequece done")

		}
	}
}
