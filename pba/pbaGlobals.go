// ----------------------NOT PERMANENENT---------------------------
package pba

import (
	"Sanntid/config"
	"Sanntid/network"
	"Sanntid/networkDriver/peers"
)

var StoredOrders [config.MElevators][config.NFloors][config.NButtons]bool
var LatestPeerList peers.PeerUpdate
var NodeStatusMap = make(map[string]network.SingleElevatorStatus)
var PreviousPrimaryID string
var AloneOnNetwork bool = false

var TakeOverInProgress bool = false
var PrimaryID string = ""
var BackupID string = ""
