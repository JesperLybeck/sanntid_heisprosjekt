package fsm

const NFloors int = 4

var PrimaryID string = ""
var BackupID string = ""

type Status struct {
	ID     string
	Orders [NFloors][3]bool
}