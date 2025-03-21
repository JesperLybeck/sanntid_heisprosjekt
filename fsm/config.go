package fsm

import (
	"Network-go/network/peers"
	"Sanntid/elevio"
	"time"
)

const NFloors int = 4
const NButtons int = 3
const MElevators int = 3

var TakeOverInProgress bool = false
var PrimaryID string = ""
var BackupID string = ""
var StartingAsPrimary bool = false
var Version int = 0
var StoredOrders [MElevators][NFloors][NButtons]bool
var IpToIndexMap = make(map[string]int)
var LatestPeerList peers.PeerUpdate
var NodeStatusMap = make(map[string]SingleElevatorStatus)
var PreviousPrimaryID string
var OrderTimeout time.Duration = 7
var AloneOnNetwork bool = false

type Status struct {
	TransmitterID string
	ReceiverID    string
	Orders        [MElevators][NFloors][NButtons]bool
	Version       int
	Map           map[string]int
	Peerlist      peers.PeerUpdate
}

type Order struct {
	ButtonEvent elevio.ButtonEvent
	ID          string
	TargetID    string
	Orders      [NFloors][NButtons]bool
}
type SingleElevatorStatus struct {
	ID             string
	PrevFloor      int
	MotorDirection elevio.MotorDirection
	Orders         [NFloors][NButtons]bool
}
type CostTuple struct {
	Cost int
	ID   string
}
type LightUpdate struct {
	LightArray [NFloors][NButtons]bool
	ID         string
}
