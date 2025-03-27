package elevator

import (
	"Sanntid/config"
	"time"
)

type ElevatorState int

const (
	Idle ElevatorState = iota
	MovingBetweenFloors
	MovingPassingFloor
	DoorOpen
)

type ElevatorOutput struct {
	MotorDirection     MotorDirection
	prevMotorDirection MotorDirection
	Door               bool
	LocalOrders        [config.NFloors][config.NButtons]bool
}

type ElevatorInput struct {
	LocalRequests      [config.NFloors][config.NButtons]bool
	PrevFloor          int
	LastClearedButtons []ButtonEvent
}
type Elevator struct {
	State              ElevatorState
	Input              ElevatorInput
	Output             ElevatorOutput
	DoorTimer          *time.Timer
	OrderCompleteTimer *time.Timer
	ObstructionTimer   *time.Timer
	DoorObstructed     bool
}

type DirectionStatePair struct {
	Direction  MotorDirection
	State      ElevatorState
	ExtraTimer bool
}
