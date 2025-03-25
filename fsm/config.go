package fsm

import (
	"Network-go/network/peers"
	"Sanntid/elevio"
	"time"
	//"golang.org/x/text/message"
)

const NFloors int = 4
const NButtons int = 3
const MElevators int = 3


var PrimaryID string = ""
var BackupID string = ""

var StartingAsPrimary bool = false
var Version int = 0
var StoredOrders [MElevators][NFloors][NButtons]bool
var IpToIndexMap = make(map[string]int)
var LatestPeerList peers.PeerUpdate
var NodeStatusMap = make(map[string]SingleElevatorStatus)
var PreviousPrimaryID string
var OrderTimeout time.Duration = 5
var AloneOnNetwork bool = false
var LastMessagesMap = make(map[string]int)

type Status struct {
	TransmitterID string
	ReceiverID    string
	Orders        [MElevators][NFloors][NButtons]bool
	Map           map[string]int
	Peerlist      peers.PeerUpdate
	StatusID      int
}

type Election struct{
	TakeOverInProgress bool
	LostNodeID string
	PrimaryID string
	BackupID string
}

type Request struct {
	ButtonEvent elevio.ButtonEvent
	ID          string
	TargetID    string
	Orders      [NFloors][NButtons]bool
	RequestID   int
}

type Order struct {
	ButtonEvent         elevio.ButtonEvent
	ResponisbleElevator string
	OrderID             int
}

type SingleElevatorStatus struct {
	ID             string
	PrevFloor      int
	MotorDirection elevio.MotorDirection
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
