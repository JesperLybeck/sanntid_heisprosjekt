package elevio

import "time"

var timer_end_time time.Time 
var timer_active bool


func Timer_start(duration time.Duration) { // starter en timer på duration
	timer_end_time = time.Now().Add(duration)
	timer_active = true
}

func Timer_stop() {
	timer_active = false
}

func Timer_timed_out() bool { // returnerer true hvis tiden er ute

	return timer_active && time.Now().After(timer_end_time)

}



// eksempel på kall: timer.Timer_start(5 * time.Second)