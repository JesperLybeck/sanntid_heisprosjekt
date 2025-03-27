package pba

import (
	"Sanntid/config"
	"time"

	"Sanntid/network"

	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/peers"
)

var LatestStatusFromPrimary network.Status

func Backup(ID string, backupSignal <-chan bool, primaryTakeover chan<- network.Takeover) {
	// Set timeout duration
	for {
		print("Waiting to become backup")
		<-backupSignal

		var primaryStatusRX = make(chan network.Status)
		var peerUpdateRX = make(chan peers.PeerUpdate)
		var backupStoredOrders = [config.MElevators][config.NFloors][config.NButtons]bool{}
		var backupNodeStatusMap = make(map[string]network.SingleElevatorStatus)
		var backupLatestPeerList peers.PeerUpdate

		go bcast.Receiver(13055, primaryStatusRX)
		go peers.Receiver(12055, peerUpdateRX)
		print("Backup")
		time.Sleep(2 * time.Second)
		primID := ""
		

	backupLoop:
		for {
			select {
			case p := <-primaryStatusRX:
				print("I am backup")
				backupStoredOrders = p.Orders //denne bøurde vi gjøre lokal.
				primID = p.TransmitterID
				backupNodeStatusMap = p.NodeStatusMap
			case p := <-peerUpdateRX:
				backupLatestPeerList = p
				if primInPeersLost(primID, p) {

					backupLatestPeerList = removeFromActivePeers(primID, backupLatestPeerList)
					takeover := network.Takeover{TakeOverInProgress: true,
						LostNodeID:     primID,
						BackupID:       "",
						PrimaryID:      ID,
						StoredOrders:   backupStoredOrders,
						NodeStatusMap: backupNodeStatusMap,
						LatestPeerList: backupLatestPeerList}

					primaryTakeover <- takeover
					print("Primary timed out")
					break backupLoop

				}

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
