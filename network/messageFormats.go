package network

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/networkDriver/peers"
	
)

// -------------------------------Message formats--------------------
type Status struct {
	TransmitterID string
	LatestPeerList peers.PeerUpdate
	Orders        [config.MElevators][config.NFloors][config.NButtons]bool
	NodeStatusMap      map[string]SingleElevatorStatus
	StatusID      int
}

type SingleElevatorStatus struct {
	ID             string
	PrevFloor      int
	MotorDirection elevator.MotorDirection
	Orders         [config.NFloors][config.NButtons]bool
	StatusID       int
}

type Election struct {
	PrimaryID    string
	MergedOrders [config.MElevators][config.NFloors][config.NButtons]bool
}

type Takeover struct {
	TakeOverInProgress bool
	LostNodeID         string
	NodeStatusMap      map[string]SingleElevatorStatus
	StoredOrders       [config.MElevators][config.NFloors][config.NButtons]bool
	LatestPeerList     peers.PeerUpdate
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
