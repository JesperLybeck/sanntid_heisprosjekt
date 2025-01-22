package elevio


import (

	"time"
	"fmt"

)

var ( 
	Elevator 	elevator
	elev_output_device 		elevator_input_device
)

func Init() {
	Elevator = elevator_uninitialized()

	conLoad("elevator.con",
	conVal("doorOpenDuration_s", &Elevator.Config.DoorOpenDuration_s, "%lf"),
	conEnum("clearRequestVariant", &Elevator.Config.ClearRequestVariant,
		conMatch(CV_All),
		conMatch(CV_InDirn),
	),
)
	output_device = elev_output_device()
}


func set_all_lights(Elevator elevator) {
    for floor := 0; floor < elevio.NumFloors(); floor++ {
        for btn := 0; btn < elevio.NumButtons(); btn++ {
            output_device.request_button_light(floor, btn, Elevator.Requests[floor][btn])
        }
    }
}

func Fsm_on_init_between_floors() {
    output_device.motor_direction(MD_Down)
    Elevator.direction = MD_Down
    Elevator.behaviour = eb_moving
}

func On_request_button_press(btnFloor int, btnType ButtonType) {
    fmt.Printf("\n\n%s(%d, %s)\n", "OnRequestButtonPress", btnFloor, button_to_string(btnType))
    

    switch Elevator.Behaviour {
    case Elevator.eb_door_open:
        if requests_should_clear_immediately(Elevator, btnFloor, btnType) {
            Timer_start(Elevator.config.door_open_duration)
        } else {
            Elevator.requests[btnFloor][btnType] = 1
        }
    case Elevator.eb_moving:
        Elevator.requests[btnFloor][btnType] = 1
    case Elevator.eb_idle:
        Elevator.requests[btnFloor][btnType] = 1
        pair := requests_choose_directions(elevator)
        Elevator.Dirn = pair.Dirn
        Elevator.Behaviour = pair.Behaviour
        switch pair.Behaviour {
        case Elevator.EB_DoorOpen:
            output_device.door_light(1)
            Timer_start(Elevator.config.door_open_duration)
            Elevator = requests_clear_at_current_floor(elevator)
        case Elevator.eb_moving:
            output_device.MotorDirection(Elevator.direction)
        case Elevator.eb_idle:
        }
    }

	set_all_lights(elevator)

    fmt.Printf("\nNew state:\n")
}

func On_floor_arrival(newFloor int) {
    fmt.Printf("\n\n%s(%d)\n", "OnFloorArrival", newFloor)

    Elevator.floor = newFloor

    output_device.floor_indicator(Elevator.floor)

    switch Elevator.behaviour {
    case Elevator.eb_moving:
        if requests_should_stop(elevator) {
            output_device.motor_direction(MD_Stop)
            output_device.door_light(1)
            elevator = requests_clear_at_current_floor(elevator)
            Timer_start(elevator.config.door_open_duration)
            set_all_lights(elevator)
            Elevator.behaviour = Elevator.eb_door_open
        }
    default:
    }

    fmt.Printf("\nNew state:\n")
    
}


func On_door_timeout() {
    fmt.Printf("\n\n%s()\n", "OnDoorTimeout")

    switch Elevator.Behaviour {
    case Elevator.eb_door_open:
        pair := requests_choose_direction(elevator)
        Elevator.direction = pair.direction
        Elevator.behaviour = pair.behaviour

        switch Elevator.behaviour {
        case Elevator.eb_door_open:
            Timer_start(Elevator.config.door_open_duration)
            elevator = requests_clear_at_current_floor(elevator)
            set_all_lights(elevator)
        case Elevator.eb_moving, Elevator.eb_idle:
            output_device.door_light(0)
            output_device.MotorDirection(Elevator.direction)
        }
    default:
    }

    fmt.Printf("\nNew state:\n")

}