package fsm

import (
	"Network-go/network/peers"
	"Sanntid/elevator"
	"time"
	//"golang.org/x/text/message"
)


//-------------------------------Message formats--------------------
type Status struct {
	TransmitterID string
	ReceiverID    string
	Orders        [MElevators][NFloors][NButtons]bool
	Map           map[string]int
	StatusID      int
}

type Election struct {
	TakeOverInProgress bool
	LostNodeID         string
	PrimaryID          string
	BackupID           string
}
type Request struct {
	ButtonEvent elevator.ButtonEvent
	ID          string
	TargetID    string
	Orders      [NFloors][NButtons]bool
	RequestID   int
}

type Order struct {
	ButtonEvent         elevator.ButtonEvent
	ResponisbleElevator string
	OrderID             int
}

type SingleElevatorStatus struct {
	ID             string
	PrevFloor      int
	MotorDirection elevator.MotorDirection
	Orders         [NFloors][NButtons]bool
	StatusID       int
}
type CostTuple struct {
	Cost int
	ID   string
}
type LightUpdate struct {
	LightArray [NFloors][NButtons]bool
	ID         string
}
