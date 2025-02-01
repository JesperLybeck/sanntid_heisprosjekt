package elevio

type Elevator_event_channels struct { //struct containing channels for state transitioning events.
	New_order     chan Elevator_order
	Door_timeout  chan struct{}
	Floor_reached chan int
	Door_open     chan struct{}
}

type Elevator_state int

const (
	Idle Elevator_state = iota
	Moving
	DoorOpen
	Maintenance
)

type Elevator_state_machine struct {
	Id             int
	Event_channels Elevator_event_channels
	Elevator_state Elevator_state
	Input_device   Elev_input_device
	Output_device  Elev_output_device
	Target_floor   int
	Order_queue    []Elevator_order
}

type Elevator_order struct {
	floor int
}

const NUMFLOORS = 4
