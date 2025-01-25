package main

import (
	"Driver-go/elevio"
)

func create_elevator_state_machine(id int) Elevator_state_machine { //lager et objekt av typen elevator_state_machine
	var elevator_fsm Elevator_state_machine
	elevator_fsm.id = id
	elevator_fsm.output_device = elevio.GetOutputDevice()
	elevator_fsm.input_device = elevio.GetInputDevice()

	elevator_fsm.event_channels.New_order = make(chan struct{})
	elevator_fsm.event_channels.Door_timeout = make(chan struct{})
	elevator_fsm.event_channels.Floor_reached = make(chan struct{})

	return elevator_fsm

}

func initialize_elevator(elevator_fsm *Elevator_state_machine) {

	elevator_fsm.output_device.Motor_direction(elevio.Direction_down) //motor direction down
	elevator_fsm.elevator_state = Moving                              //sets state to moving
	println("going up")

	<-elevator_fsm.event_channels.Floor_reached //listening for floot reached

	elevator_fsm.output_device.Motor_direction(elevio.Direction_stop) //stop motor
	elevator_fsm.elevator_state = DoorOpen                            //change elevator state to idle
	println("start floor reached")

}

func (elevator_fsm *Elevator_state_machine) handleNewOrder() {
	println("new order")
	elevator_fsm.elevator_state = Moving
}

func (elevator_fsm *Elevator_state_machine) handleDoorTimeout() {
	println("door timeout")
	elevator_fsm.elevator_state = Idle //hva blir riktig state her?
}

func (elevator_fsm *Elevator_state_machine) handleFloorReached() {
	println("floor reached")
	elevator_fsm.elevator_state = DoorOpen

}

func run_elevator(elevator_fsm *Elevator_state_machine) {

	for {
		select {
		case <-elevator_fsm.event_channels.New_order:
			elevator_fsm.handleNewOrder()

		case <-elevator_fsm.event_channels.Door_timeout:
			elevator_fsm.handleDoorTimeout()

		case <-elevator_fsm.event_channels.Floor_reached:
			elevator_fsm.handleFloorReached()

		}

	}
}
