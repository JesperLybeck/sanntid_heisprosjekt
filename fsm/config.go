package fsm

import "Sanntid/elevio"

const NFloors int = 4
const NButtons int = 3
const MElevators int = 3

var PrimaryID string = ""
var BackupID string = ""
var StartingAsPrimary bool = false
var Version int = 0
var StoredOrders [NFloors][NButtons][MElevators]bool
var IpToIndexMap = make(map[string]int)

type Status struct {
	TransmitterID string
	ReceiverID    string
	Orders        [NFloors][NButtons][MElevators]bool
	Version       int
	Map           map[string]int
}

type Order struct {
	ButtonEvent elevio.ButtonEvent
	ID          string
	TargetID    string
	Orders      [NFloors][3]bool
}
