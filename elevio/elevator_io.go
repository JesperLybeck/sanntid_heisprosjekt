package elevio

import (
	"fmt"
	"net"
	"sync"
	"time"
)
import (
	"fmt"
	"net"
	"sync"
	"time"
)

const _pollRate = 20 * time.Millisecond

var _initialized bool = false
var _numFloors int = 4
var _mtx sync.Mutex
var _conn net.Conn
var _initialized bool = false
var _numFloors int = 4
var _mtx sync.Mutex
var _conn net.Conn

type motor_direction int

const (
	direction_up   motor_direction = 1
	direction_down                 = -1
	direction_stop                 = 0
)

type button int

const (
	button_hall_up   button = 0
	button_hall_down        = 1
	button_cab              = 2
)

type button_event struct {
	floor  int
	button button
}
type elev_input_device struct {
	floor_sensor   func() int
	request_button func(button, int) bool
	stop_button    func() bool
	obstruction    func() bool
}

type elev_output_device struct {
	floor_indicator      func(int)
	request_button_light func(button, int, bool)
	door_light           func(bool)
	stop_button_light    func(bool)
	motor_direction      func(motor_direction)
}

func Init(addr string, numFloors int) {
	if _initialized {
		fmt.Println("Driver already initialized!")
		return
	}
	_numFloors = numFloors
	_mtx = sync.Mutex{}
	var err error
	_conn, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	_initialized = true
}

func Set_motor_direction(dir motor_direction) {
	write([4]byte{1, byte(dir), 0, 0})
}

func Set_button_lamp(button button, floor int, value bool) {
	write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

func Set_floor_indicator(floor int) {
	write([4]byte{3, byte(floor), 0, 0})
}

func Set_door_open_lamp(value bool) {
	write([4]byte{4, toByte(value), 0, 0})
}

func Set_stop_lamp(value bool) {
	write([4]byte{5, toByte(value), 0, 0})
}

func Poll_buttons(receiver chan<- button_event) {
	prev := make([][3]bool, _numFloors)
	for {
		time.Sleep(_pollRate)
		for f := 0; f < _numFloors; f++ {
			for b := button(0); b < 3; b++ {
				v := Get_button(b, f)
				if v != prev[f][b] && v != false {
					receiver <- button_event{f, button(b)}
				}
				prev[f][b] = v
			}
		}
	}
}

func Poll_floor_sensor(receiver chan<- int) {
	prev := -1
	for {
		time.Sleep(_pollRate)
		v := Get_floor()
		if v != prev && v != -1 {
			receiver <- v
		}
		prev = v
	}
}

func Poll_stop_button(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := Get_stop()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func Poll_obstruction_switch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := Get_obstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func Get_button(button button, floor int) bool {
	a := read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(a[1])
}

func Get_floor() int {
	a := read([4]byte{7, 0, 0, 0})
	if a[1] != 0 {
		return int(a[2])
	} else {
		return -1
	}
}

func Get_stop() bool {
	a := read([4]byte{8, 0, 0, 0})
	return toBool(a[1])
}

func Get_obstruction() bool {
	a := read([4]byte{9, 0, 0, 0})
	return toBool(a[1])
}

func read(in [4]byte) [4]byte {
	_mtx.Lock()
	defer _mtx.Unlock()


	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = _conn.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func write(in [4]byte) {
	_mtx.Lock()
	defer _mtx.Unlock()


	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
}

func toByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

func toBool(a byte) bool {
	var b bool = false
	if a != 0 {
		b = true
	}
	return b
}

// ElevioDirnToString converts a Dirn to its string representation
func Direction_to_string(d motor_direction) string {
	switch d {
	case direction_up:
		return "Direction_Up"
	case direction_down:
		return "Direction_Down"
	case direction_stop:
		return "Direction_Stop"
	default:
		return "D_UNDEFINED"
	}
}

// ElevioButtonToString converts a Button to its string representation
func button_to_string(b button) string {
	switch b {
	case button_hall_up:
		return "button_hall_up"
	case button_hall_down:
		return "button_hall_down"
	case button_cab:
		return "button_cab"
	default:
		return "button_UNDEFINED"
	}
}

func GetInputDevice() elev_input_device {
	return elev_input_device{
		floor_sensor:   Get_floor,
		request_button: Get_button,
		stop_button:    Get_stop,
		obstruction:    Get_obstruction,
	}
}

func GetOutputDevice() elev_output_device {
	return elev_output_device{
		floor_indicator:      Set_floor_indicator,
		request_button_light: Set_button_lamp,
		door_light:           Set_door_open_lamp,
		stop_button_light:    Set_stop_lamp,
		motor_direction:      Set_motor_direction,
	}
}
