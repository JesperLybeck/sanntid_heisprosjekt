package network

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/networkDriver/peers"
)

type Status struct {
	TransmitterID      string
	Orders             [config.MElevators][config.NFloors][config.NButtons]bool
	StatusID           int
	AloneOnNetwork     bool
	TakeOverInProgress bool
}
type SingleElevatorStatus struct {
	ID             string
	PrevFloor      int
	MotorDirection elevator.MotorDirection
	Orders         [config.NFloors][config.NButtons]bool
	StatusID       int
}
type Takeover struct {
	StoredOrders       [config.MElevators][config.NFloors][config.NButtons]bool
	PreviousPrimaryID  string
	Peerlist           peers.PeerUpdate
	NodeMap            map[string]SingleElevatorStatus
	TakeOverInProgress bool
}
type Election struct {
	PrimaryID    string
	MergedOrders [config.MElevators][config.NFloors][config.NButtons]bool
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

type LightUpdate struct {
	LightArray [config.NFloors][config.NButtons]bool
	ID         string
}
