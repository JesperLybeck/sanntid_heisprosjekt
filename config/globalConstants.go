package config

import "time"

const NFloors int = 4
const NButtons int = 3
const MElevators int = 3

//timer constants //Her må vi sette alle timere konstantene for hele systemet.
const OrderTimeout time.Duration = 9
const DoorTimeout time.Duration = 3
const ObstructionTimeout time.Duration = 9

var IDToIndexMap = map[string]int{

	"111": 0,
	"222": 1,
	"333": 2,
}
