package elevio

type elevator_behaviour int

const (
	eb_idle elevator_behaviour = iota
	eb_door_open
	eb_moving
)

type clear_requests_variant int

const (
	cv_all clear_requests_variant = iota
	cv_in_dirn
)

type config struct {
	clear_requests_variant clear_requests_variant
	door_open_duration int
}

type elevator struct {
	floor int
	direction motor_direction
	requests [N_FLOORS][N_BUTTONS]bool
	behaviour elevator_behaviour
	config config
}

func Elevator_uninitialized() elevator {
	return elevator{
		floor: -1,
		direction: direction_stop,
		behaviour: eb_idle,
		config: config{
            clear_requests_variant: cv_all,
            door_open_duration:     3.0,
        },
	}
}