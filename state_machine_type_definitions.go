package main

import "Driver-go/elevio"

type Elevator_event_channels struct { //struct containing channels for state transitioning events.
	New_order     chan struct{}
	Door_timeout  chan struct{}
	Floor_reached chan struct{}
}

type Elevator_state int

const (
	Idle Elevator_state = iota
	Moving
	DoorOpen
	Maintenance
)

type Elevator_state_machine struct {
	id             int
	event_channels Elevator_event_channels
	elevator_state Elevator_state
	input_device   elevio.Elev_input_device
	output_device  elevio.Elev_output_device
}

type Elevator_order struct {
	floor int
}
