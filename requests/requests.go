package requests


type dirn_behaviour_pair struct {
	direction int
	behaviour behaviour
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
		if elevator.requests[elevator.Floor][i]{
			return true
			
		}
	return false
	}
}
func Request_choose_directions(elevator elevator)dirn_behaviour_pair {
	switch elevator.direction {
	case DIRN_UP:
		if Requests_above(elevator) {
			return dirn_behaviour_pair{d_up, BEHAVIOUR_MOVING}
		} else if Requests_below(elevator) {
			return dirn_behaviour_pair{d_down, BEHAVIOUR_MOVING}
		} else if Requests_here(elevator) {
			return dirn_behaviour_pair{d_stop, BEHAVIOUR_OPEN}
		} else {
			return dirn_behaviour_pair{d_stop, BEHAVIOUR_IDLE}
		}
	case DIRN_DOWN:
		if Requests_below(elevator) {
			return dirn_behaviour_pair{d_down, BEHAVIOUR_MOVING}
		} else if Requests_above(elevator) {
			return dirn_behaviour_pair{d_up, BEHAVIOUR_MOVING}
		} else if Requests_here(elevator) {
			return dirn_behaviour_pair{d_stop, BEHAVIOUR_OPEN}
		} else {
			return dirn_behaviour_pair{d_stop, BEHAVIOUR_IDLE}
		}
	case DIRN_STOP:
		if Requests_below(elevator) {
			return dirn_behaviour_pair{d_down, behaviour_moving}
		}
		if Requests_above(elevator) {
			return dirn_behaviour_pair{d_up, behaviour_moving}
		}
		if Requests_here(elevator) {
			return dirn_behaviour_pair{d_stop, behaviour_open}
		}
		return dirn_behaviour_pair{d_stop, behaviour_idle}
	default:
		return dirn_behaviour_pair{d_stop, behaviour_idle}
		
	}
}

func Requests_should_stop(elevator elevator) bool {
	switch elevator.direction {
	case d_up:
		return elevator.requests[elevator.floor][b_hall_up] || elevator.requests[elevator.floor][b_cab] || !Requests_above(elevator)
	case d_down:
		return elevator.requests[elevator.floor][b_hall_down] || elevator.requests[elevator.floor][b_cab] || !Requests_below(elevator)
	case d_stop:
	default:
		return false
	}
}

func Requests_should_clear_immediately(elevator elevator, btn_floor int, btn_type button) bool {
	switch elevator.config.clear_requests_variant {
	case cv_all:
		return elevator.floor == btn_floor
	case cv_in_dirn:
		return elevator.floor == btn_floor && 
		(elevator.direction == d_up &&
		btn_type == b_hallup) ||
		(elevator.direction == d_down &&
		btn_type == b_halldown) ||
		elevator.direction == d_stop ||
		btn_type == b_cab
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
		elevator.requests[elevator.floor][b_cab] = false
		switch elevator.direction {
		case d_up:
			if !Requests_above(elevator) {
				elevator.requests[elevator.floor][b_hall_down] = false
			}
			elevator.requests[elevator.floor][b_hall_up] = false
			break;
		case d_down:
			if !Requests_below(elevator) {
				elevator.requests[elevator.floor][b_hall_up] = false
			}
			elevator.requests[elevator.floor][b_hall_down] = false
			break;
		case d_stop:
		default:
			elevator.requests[elevator.floor][b_hall_up] = false
			elevator.requests[elevator.floor][b_hall_down] = false
			break;
		}
	default:
		break;
	}
	return elevator
}
