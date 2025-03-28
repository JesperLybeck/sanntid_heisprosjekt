package config

import "time"

const NFloors int = 4
const NButtons int = 3
const MElevators int = 3

const OrderTimeout time.Duration = 5
const DoorTimeout time.Duration = 3
const ObstructionTimeout time.Duration = 7

var IDToIndexMap = map[string]int{

	"111": 0,
	"222": 1,
	"333": 2,
}
