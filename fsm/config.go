package fsm

import "Sanntid/elevio"

const NFloors int = 4

var PrimaryID string = ""
var BackupID string = ""
var StartingAsPrimary bool = false

type Status struct {
	TransmitterID string
	RecieverID    string
	Orders        [NFloors][3]bool
}

type Order struct {
	elevio.ButtonEvent
	ID       string
	TargetID string
	Orders   [NFloors][3]bool
	Role     string
}
