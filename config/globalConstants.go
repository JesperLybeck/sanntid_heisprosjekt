package config

import "time"

const NFloors int = 4
const NButtons int = 3
const MElevators int = 3

const OrderTimeout time.Duration = 7


var IpToIndexMap = map[string]int{
	"11": 0,
	"22": 1,
	"33": 2,
}
