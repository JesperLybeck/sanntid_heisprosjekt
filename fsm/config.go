package fsm

import "Sanntid/elevio"

const NFloors int = 4

var PrimaryID string = ""
var BackupID string = ""

type Status struct {
	ID     string
	Orders [NFloors][3]bool
	Role   string
}

type Order struct {
	elevio.ButtonEvent
	ID       string
	TargetID string
	Orders   [NFloors][3]bool
	Role     string
}
