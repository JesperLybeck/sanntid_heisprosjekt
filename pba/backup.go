package pba

import (
	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/peers"
)

func Backup(ID string, primaryElection <-chan network.Election, done chan<- network.Takeover) {
	var primaryStatusRX = make(chan network.Status)
	var peerUpdateRX = make(chan peers.PeerUpdate)
	var nodeStatusUpdateRX = make(chan network.SingleElevatorStatus)

	var latestStatusFromPrimary network.Status
	var latestPeerList peers.PeerUpdate
	var primaryID string
	var previousPrimaryID string
	var nodeMap = make(map[string]network.SingleElevatorStatus)

	go bcast.Receiver(13055, primaryStatusRX)
	go peers.Receiver(12055, peerUpdateRX)
	go bcast.Receiver(13059, nodeStatusUpdateRX)

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

func removeFromActivePeers(ID string, peerlist peers.PeerUpdate) peers.PeerUpdate {
	newPeerList := make([]string, 0)
	lostPeers := make([]string, 0)
	for i := 0; i < len(peerlist.Peers); i++ {
		if peerlist.Peers[i] != ID {
			newPeerList = append(newPeerList, peerlist.Peers[i])
		} else {
			lostPeers = append(lostPeers, peerlist.Peers[i])
		}
		for i := 0; i < len(peerlist.Lost); i++ {
			lostPeers = append(lostPeers, peerlist.Lost[i])
		}
	}
	return peers.PeerUpdate{Peers: newPeerList, Lost: lostPeers, New: peerlist.New}
}
