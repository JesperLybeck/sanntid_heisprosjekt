package elevio


import (

	"fmt"
    "time"

)

var ( 
	Elevator elevator
	output_device elev_output_device 		
)

func Init() {
	Elevator = Elevator_uninitialized()

	conLoad("elevator.con",
	conVal("doorOpenDuration_s", &Elevator.config.door_open_duration, "lf"),
	conEnum("clearRequestVariant", &Elevator.config.clear_requests_variant,
		conMatch(CV_All),
		conMatch(CV_InDirn),
	),
)
	output_device = GetOutputDevice()
}


func set_all_lights(Elevator elevator) {
    for floor := 0; floor < N_FLOORS; floor++ {
        for btn := 0; btn < N_BUTTONS; btn++ {
            output_device.request_button_light(button(Elevator.floor), btn, Elevator.requests[floor][btn])
        }
    }
}

func Fsm_on_init_between_floors() {
    output_device.motor_direction(direction_down)
    Elevator.direction = direction_down
    Elevator.behaviour = eb_moving
}

func On_request_button_press(btnFloor int, btnType button) {
    fmt.Printf("\n\n%s(%d, %s)\n", "OnRequestButtonPress", btnFloor, button_to_string(btnType))
    

    switch Elevator.behaviour {
    case eb_door_open:
        if Requests_should_clear_immediately(Elevator, btnFloor, btnType) {
            Timer_start(time.Duration(Elevator.config.door_open_duration))
        } else {
            Elevator.requests[btnFloor][btnType] = true
        }
    case eb_moving:
        Elevator.requests[btnFloor][btnType] = true
    case eb_idle:
        Elevator.requests[btnFloor][btnType] = true
        pair := Request_choose_directions(Elevator)
        Elevator.direction = pair.direction
        Elevator.behaviour = pair.behaviour
        switch pair.behaviour {
        case eb_door_open:
            output_device.door_light(true)
            Timer_start(time.Duration(Elevator.config.door_open_duration))
            Elevator = Requests_clear_at_current_floor(Elevator)
        case eb_moving:
            output_device.motor_direction(Elevator.direction)
        case eb_idle:
        }
    }

	set_all_lights(Elevator)

    fmt.Printf("\nNew state:\n")
}

func On_floor_arrival(newFloor int) {
    fmt.Printf("\n\n%s(%d)\n", "OnFloorArrival", newFloor)

    Elevator.floor = newFloor

    output_device.floor_indicator(Elevator.floor)

    switch Elevator.behaviour {
    case eb_moving:
        if Requests_should_stop(Elevator) {
            output_device.motor_direction(direction_stop)
            output_device.door_light(true)
            Elevator = Requests_clear_at_current_floor(Elevator)
            Timer_start(time.Duration(Elevator.config.door_open_duration))
            set_all_lights(Elevator)
            Elevator.behaviour = eb_door_open
        }
    default:
    }

    fmt.Printf("\nNew state:\n")
    
}


func On_door_timeout() {
    fmt.Printf("\n\n%s()\n", "OnDoorTimeout")

    switch Elevator.behaviour {
        case eb_door_open:
            Timer_start(time.Duration(Elevator.config.door_open_duration))
            Elevator = Requests_clear_at_current_floor(Elevator)
            set_all_lights(Elevator)
        case eb_moving, eb_idle:
            output_device.door_light(false)
            output_device.motor_direction(Elevator.direction)
        }
        fmt.Printf("\nNew state:\n")
    }
