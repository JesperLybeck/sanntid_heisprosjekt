package config

import (
	"Network-go/network/peers"
	"time"
)

// --------------------------------CONSTANTS--------------------------------
const NFloors int = 4
const NButtons int = 3
const MElevators int = 3

var TakeOverInProgress bool = false
var PrimaryID string = ""
var BackupID string = ""
var StartingAsPrimary bool = false
var StoredOrders [MElevators][NFloors][NButtons]bool
var IpToIndexMap = make(map[string]int)
var LatestPeerList peers.PeerUpdate
var NodeStatusMap = make(map[string]SingleElevatorStatus)
var PreviousPrimaryID string
var OrderTimeout time.Duration = 5
var AloneOnNetwork bool = false
var LastMessagesMap = make(map[string]int)
