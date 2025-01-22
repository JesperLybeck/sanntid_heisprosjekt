package elevio


type dirn_behaviour_pair struct {
	direction motor_direction
	behaviour elevator_behaviour
}

func Requests_above(elevator elevator) bool {
	for i := elevator.floor + 1; i < N_FLOORS; i++ {
		if elevator.requests[i][0] || elevator.requests[i][1] || elevator.requests[i][2] {
			return true
		}
	}
	return false
}

func Requests_below(elevator elevator) bool {
	for i := 0; i < elevator.floor; i++ {
		if elevator.requests[i][0] || elevator.requests[i][1] || elevator.requests[i][2] {
			return true
		}
	}
	return false
}

func Requests_here(elevator elevator) bool {
	for i := 0; i < N_BUTTONS; i++ {
		if elevator.requests[elevator.floor][i]{
			return true
			
		}
	
	}
	return false
}
func Request_choose_directions(elevator elevator)dirn_behaviour_pair {
	switch elevator.direction {
	case direction_up:
		if Requests_above(elevator) {
			return dirn_behaviour_pair{direction_up, eb_moving}
		} else if Requests_below(elevator) {
			return dirn_behaviour_pair{direction_down, eb_moving}
		} else if Requests_here(elevator) {
			return dirn_behaviour_pair{direction_stop, eb_door_open}
		} else {
			return dirn_behaviour_pair{direction_stop, eb_idle}
		}
	case direction_down:
		if Requests_below(elevator) {
			return dirn_behaviour_pair{direction_down, eb_moving}
		} else if Requests_above(elevator) {
			return dirn_behaviour_pair{direction_up, eb_moving}
		} else if Requests_here(elevator) {
			return dirn_behaviour_pair{direction_stop, eb_door_open}
		} else {
			return dirn_behaviour_pair{direction_stop, eb_idle}
		}
	case direction_stop:
		if Requests_below(elevator) {
			return dirn_behaviour_pair{direction_down, eb_moving}
		}
		if Requests_above(elevator) {
			return dirn_behaviour_pair{direction_up, eb_moving}
		}
		if Requests_here(elevator) {
			return dirn_behaviour_pair{direction_stop, eb_door_open}
		}
		return dirn_behaviour_pair{direction_stop, eb_idle}
	default:
		return dirn_behaviour_pair{direction_stop, eb_idle}
		
	}
}

func Requests_should_stop(elevator elevator) bool {
	switch elevator.direction {
	case direction_up:
		return elevator.requests[elevator.floor][button_hall_up] || elevator.requests[elevator.floor][button_cab] || !Requests_above(elevator)
	case direction_down:
		return elevator.requests[elevator.floor][button_hall_down] || elevator.requests[elevator.floor][button_cab] || !Requests_below(elevator)
	case direction_stop:
	default:
		
	}
	return false
}

func Requests_should_clear_immediately(elevator elevator, btn_floor int, btn_type button) bool {
	switch elevator.config.clear_requests_variant {
	case cv_all:
		return elevator.floor == btn_floor
	case cv_in_dirn:
		return elevator.floor == btn_floor && 
		(elevator.direction == direction_up &&
		btn_type == button_hall_up) ||
		(elevator.direction == direction_down &&
		btn_type == button_hall_down) ||
		elevator.direction == direction_stop ||
		btn_type == button_cab
	default:
		return false
	}
}

func Requests_clear_at_current_floor(elevator elevator) elevator {
	switch elevator.config.clear_requests_variant {
	case cv_all:
		for i := 0; i < N_BUTTONS; i++ {
			elevator.requests[elevator.floor][i] = false
		}
	case cv_in_dirn:
		elevator.requests[elevator.floor][button_cab] = false
		switch elevator.direction {
		case direction_up:
			if !Requests_above(elevator) {
				elevator.requests[elevator.floor][button_hall_down] = false
			}
			elevator.requests[elevator.floor][button_hall_up] = false
			break;
		case direction_down:
			if !Requests_below(elevator) {
				elevator.requests[elevator.floor][button_hall_up] = false
			}
			elevator.requests[elevator.floor][button_hall_down] = false
			break;
		case direction_stop:
		default:
			elevator.requests[elevator.floor][button_hall_up] = false
			elevator.requests[elevator.floor][button_hall_down] = false
			break;
		}
	default:
		break;
	}
	return elevator
}
