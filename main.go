package main

import "Driver-go/elevio"

//import "Driver-go/elevio"

func main() {
	var elevator_1 Elevator_state_machine = create_elevator_state_machine(1)
	var orders = []Elevator_order{}
	initialize_elevator(&elevator_1)
	run_elevator(&elevator_1)
	button_events_channel := make(chan elevio.Button_event)
	go elevio.Poll_buttons(button_events_channel)

	for {

		select {
		case button_press := <-button_events_channel:
			Handle_button_event(button_press, &orders)

		}

	}

}
