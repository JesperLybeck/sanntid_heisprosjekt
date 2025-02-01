package main

import "Driver-go/elevio"

func main() {
	println("hei")
	elevio.Init("localhost:15657", elevio.NUMFLOORS)
	var elevator_1 elevio.Elevator_state_machine = elevio.Create_elevator_state_machine(1)
	//var orders = []Elevator_order{}
	elevio.Initialize_elevator(&elevator_1)
	button_events_channel := make(chan elevio.Button_event)
	
	
	go elevio.Run_elevator(&elevator_1)
	go elevio.Poll_buttons(button_events_channel)
	go elevio.Poll_floor_sensor(elevator_1.Event_channels.Floor_reached)

	//hvis det hadde vært gjort på kabinett, så legges det inn i egen kø, dersom det kommer i etasjepanel så skal det bli sendt om man er IDel
	

	for {
		select {
		case button_event := <-button_events_channel:
			elevio.Handle_button_event(button_event, &elevator_1.Order_queue) //knappen blir lagt inn i heiskøen 
		}
	}

	
}
