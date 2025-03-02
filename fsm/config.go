package fsm

import "Sanntid/elevio"

const NFloors int = 4

var PrimaryID string = ""
var BackupID string = ""
var StartingAsPrimary bool = false
var LostElevators []string
var Version int = 0

type Status struct {
	TransmitterID string
	ReceiverID    string
	Orders        [NFloors][3]bool
	Version       int
}

type Order struct {
	elevio.ButtonEvent
	ID       string
	TargetID string
	Orders   [NFloors][3]bool
}
