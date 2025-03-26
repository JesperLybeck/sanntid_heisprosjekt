package network

import (
	"Sanntid/config"
	"Sanntid/elevator"
)

// -------------------------------Message formats--------------------
type Status struct {
	TransmitterID string
	ReceiverID    string
	Orders        [config.MElevators][config.NFloors][config.NButtons]bool
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
	Orders      [config.NFloors][config.NButtons]bool
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
	Orders         [config.NFloors][config.NButtons]bool
	StatusID       int
}

type LightUpdate struct {
	LightArray [config.NFloors][config.NButtons]bool
	ID         string
}
