package pba

import (
	"Sanntid/config"

	"Sanntid/network"

	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/peers"
)

var LatestStatusFromPrimary network.Status

func Backup(ID string, primaryElection chan<- network.Election) {
	// Set timeout duration
	var primaryStatusRX = make(chan network.Status)
	var peerUpdateRX = make(chan peers.PeerUpdate)

	go bcast.Receiver(13055, primaryStatusRX)
	go peers.Receiver(12055, peerUpdateRX)
	print("Backup")

	for {
		if PrimaryID != ID {

			select {

			case p := <-peerUpdateRX:
				LatestPeerList = p
				if primInPeersLost(PrimaryID, p) {

					LatestPeerList = removeFromActivePeers(PrimaryID, LatestPeerList)
					PreviousPrimaryID = PrimaryID
					PrimaryID = ID
					BackupID = ""
					TakeOverInProgress = true

				}

			case p := <-primaryStatusRX:
				//print("I am backup")
				StoredOrders = p.Orders //denne bøurde vi gjøre lokal.
				config.IpToIndexMap = p.Map

			}
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
