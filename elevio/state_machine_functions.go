package elevio

import (
	"time"
)

func Create_elevator_state_machine(id int) Elevator_state_machine { //lager et objekt av typen elevator_state_machine
	var elevator_fsm Elevator_state_machine
	elevator_fsm.Id = id
	elevator_fsm.Output_device = GetOutputDevice()
	elevator_fsm.Input_device = GetInputDevice()

	elevator_fsm.Event_channels.New_order = make(chan Elevator_order)
	elevator_fsm.Event_channels.Door_timeout = make(chan struct{})
	elevator_fsm.Event_channels.Floor_reached = make(chan int)
	elevator_fsm.Event_channels.Door_open = make(chan struct{})

	return elevator_fsm

}

func Initialize_elevator(elevator_fsm *Elevator_state_machine) {

	elevator_fsm.Output_device.Motor_direction(Direction_down) //motor direction down
	println("going up")
	for Get_floor() == -1 {
		Set_motor_direction(Direction_up)
	}
	Set_motor_direction(Direction_stop)
	println("start floor reached")
	elevator_fsm.Output_device.Door_light(true)
	//time.AfterFunc(3*time.Second, func() {
		time.Sleep(3 * time.Second)	
		println("initializing done")
		elevator_fsm.Output_device.Door_light(false)
		elevator_fsm.Elevator_state = Idle
}

func (elevator_fsm *Elevator_state_machine) handleNewOrder() {
	println("new order")
	executeOrder(Elevator_order{floor: 3}, elevator_fsm)
	elevator_fsm.Elevator_state = Moving
}

func (elevator_fsm *Elevator_state_machine) handleFloorReached() {
	

	if elevator_fsm.Target_floor == Get_floor() {
		//Send på beskjed til master at man har nådd etasje
		elevator_fsm.Output_device.Motor_direction(Direction_stop)
		time.Sleep(1 * time.Second)
		go func ()  {elevator_fsm.Event_channels.Door_open <- struct{}{}}() 
		println("check if order is at this floor")
		// Skal sjekke om etasjen er lik ordre etasje, hvis ja, åpne døren
		// Hvis ikke, fortsett å kjøre
	}
}

func (elevator_fsm *Elevator_state_machine) handleDoorOpen() {
	println("door open")
	elevator_fsm.Output_device.Door_light(true)

	time.AfterFunc(3*time.Second, func() {
		go func (){elevator_fsm.Event_channels.Door_timeout <- struct{}{}}()
	})
}

func (elevator_fsm *Elevator_state_machine) handleDoorTimeout() {
	println("door timeout")
	elevator_fsm.Output_device.Door_light(false)
	elevator_fsm.Elevator_state = Idle
}

func executeOrder(order Elevator_order, elevator_fsm *Elevator_state_machine) {
	//kjører til ordre etasje
	current_floor := Get_floor()
	elevator_fsm.Target_floor = order.floor
	if current_floor < order.floor {
		elevator_fsm.Output_device.Motor_direction(Direction_up)
	} else if current_floor > order.floor {
		elevator_fsm.Output_device.Motor_direction(Direction_down)
	} else {
		println("Already at target floor")
		go func (){elevator_fsm.Event_channels.Floor_reached <- current_floor}()
	}
	
	
}

func Run_elevator(elevator_fsm *Elevator_state_machine) {

	for {
		select {
		case <-elevator_fsm.Event_channels.New_order:
			elevator_fsm.handleNewOrder()

		case <-elevator_fsm.Event_channels.Door_timeout:
			elevator_fsm.handleDoorTimeout()

		case <-elevator_fsm.Event_channels.Floor_reached:
			println("floor reached")
			elevator_fsm.handleFloorReached()

		case <-elevator_fsm.Event_channels.Door_open:
			println("door open")
			elevator_fsm.handleDoorOpen()

		}

	}
}
