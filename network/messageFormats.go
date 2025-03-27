package network

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/networkDriver/peers"
)

// -------------------------------Message formats--------------------
type Status struct {
	TransmitterID      string
	Orders             [config.MElevators][config.NFloors][config.NButtons]bool
	StatusID           int
	PreviousPrimaryID  string
	AloneOnNetwork     bool
	TakeOverInProgress bool
	PeerList           peers.PeerUpdate
}

type Election struct {
	TakeOverInProgress bool
	LostNodeID         string
	PrimaryID          string
	BackupID           string
	MergedOrders       [config.MElevators][config.NFloors][config.NButtons]bool
}
type Request struct {
	ButtonEvent elevator.ButtonEvent
	ID          string
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
