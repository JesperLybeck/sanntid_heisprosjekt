package pba

import (
	"Sanntid/config"

	"Sanntid/network"

	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/peers"
)

var LatestStatusFromPrimary network.Status

func Backup(ID string, primaryElection <-chan network.Election, done chan<- network.Takeover) {
	// Set timeout duration
	var primaryStatusRX = make(chan network.Status)
	var peerUpdateRX = make(chan peers.PeerUpdate)
	var nodeStatusUpdateRX = make(chan network.SingleElevatorStatus)
	var latestStatusFromPrimary network.Status
	var latestPeerList peers.PeerUpdate
	var primaryID string
	var previousPrimaryID string
	var nodeMap map[string]network.SingleElevatorStatus
	//var takeOverInProgress bool

	go bcast.Receiver(13055, primaryStatusRX)
	go peers.Receiver(12055, peerUpdateRX)
	go bcast.Receiver(13056, nodeStatusUpdateRX)
	//print("i am backup")

	for {

		select {
		case p := <-primaryElection:

			if p.PrimaryID == ID {
				takeoverState := network.Takeover{
					StoredOrders:       latestStatusFromPrimary.Orders,
					PreviousPrimaryID:  previousPrimaryID,
					Peerlist:           latestPeerList,
					NodeMap:            nodeMap,
					TakeOverInProgress: false,
				}
				//go Primary(ID, primaryElection, takeoverState, done)
				done <- takeoverState
				return
			}
		case n := <-nodeStatusUpdateRX:
			nodeMap = UpdateNodeMap(n.ID, n, nodeMap)

		case p := <-peerUpdateRX:

			latestPeerList = p
			if primInPeersLost(primaryID, p) {

				latestPeerList = removeFromActivePeers(primaryID, latestPeerList)
				previousPrimaryID = primaryID

				takeoverState := network.Takeover{
					StoredOrders:       latestStatusFromPrimary.Orders,
					PreviousPrimaryID:  previousPrimaryID,
					Peerlist:           latestPeerList,
					NodeMap:            nodeMap,
					TakeOverInProgress: true,
				}

				//go Primary(ID, primaryElection, takeoverState)
				done <- takeoverState
				return

			}

		case p := <-primaryStatusRX:
			latestStatusFromPrimary = p
			primaryID = p.TransmitterID

		}

	}

}

func primInPeersLost(primID string, peerUpdate peers.PeerUpdate) bool {
	for i := 0; i < len(peerUpdate.Lost); i++ {
		if peerUpdate.Lost[i] == primID {
			return true
		}
	}
	return false
}

func mergeOrders(orders1 [config.MElevators][config.NFloors][config.NButtons]bool, orders2 [config.MElevators][config.NFloors][config.NButtons]bool) [config.MElevators][config.NFloors][config.NButtons]bool {
	var mergedOrders [config.MElevators][config.NFloors][config.NButtons]bool

	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons-1; j++ {
			for k := 0; k < config.MElevators; k++ {
				if orders1[k][i][j] || orders2[k][i][j] {
					mergedOrders[k][i][j] = true
				}
			}
		}
	}
	return mergedOrders
}

func removeFromActivePeers(ID string, peerlist peers.PeerUpdate) peers.PeerUpdate {

	newPeerList := make([]string, 0)
	lostPeers := make([]string, 0)
	for i := 0; i < len(peerlist.Peers); i++ {
		if peerlist.Peers[i] != ID {

			newPeerList = append(newPeerList, peerlist.Peers[i]) //kopierer over alle noder som ikke skal fjernes i new peer list
		} else {
			lostPeers = append(lostPeers, peerlist.Peers[i])

		}
		for i := 0; i < len(peerlist.Lost); i++ {

			lostPeers = append(lostPeers, peerlist.Lost[i]) //kopierer over de andre på lost peers
		}

	}
	return peers.PeerUpdate{Peers: newPeerList, Lost: lostPeers, New: peerlist.New}

}
