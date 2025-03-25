package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/fsm"
	"fmt"
	"time"
)

var LatestStatusFromPrimary fsm.Status

func Backup(ID string, primaryElection chan<- fsm.Election) {
	var timeout = time.After(2 * time.Second) // Set timeout duration
	var primaryStatusRX = make(chan fsm.Status)
	go bcast.Receiver(13055, primaryStatusRX)
	print("Backup")
	
	for {
		if fsm.BackupID == ID {

			select {
			case p := <-primaryStatusRX:
				//print("I am backup")
				fsm.StoredOrders = p.Orders
				fsm.IpToIndexMap = p.Map
				fsm.LatestPeerList = p.Peerlist

				timeout = time.After(2 * time.Second)

			case <-timeout:
				fmt.Println("Primary timed out")
				fsm.LatestPeerList = removeFromActivePeers(fsm.PrimaryID, fsm.LatestPeerList)
				fmt.Print("LatestPeerlist from prim timeout",fsm.LatestPeerList)
				fmt.Println("New peerlist", fsm.LatestPeerList)
				fsm.Version++
				fsm.PreviousPrimaryID = fsm.PrimaryID
				fsm.PrimaryID = ID
				fsm.BackupID = ""
				fsm.TakeOverInProgress = true
				

			}
		}
	}

}

/*

func Backup(ID string) {
	var timeout = time.After(2 * time.Second) // Set timeout duration
	var primaryStatusRX = make(chan fsm.Status)
	go bcast.Receiver(13055, primaryStatusRX)
	LatestStatusFromPrimary := fsm.Status{}
	isBackup := false
	for {
		if !isBackup {
			select {
			case p := <-primaryStatusRX:

				//fmt.Println("Backup received from primary")

				if fsm.PrimaryID == ID && p.TransmitterID != ID {
					intID, _ := strconv.Atoi(ID[len(ID)-2:])
					intTransmitterID, _ := strconv.Atoi(p.TransmitterID[len(ID)-2:])
					//Her mottar en primary melding fra en annen primary
					// Dette er bad med mye pakketap
					print("MyID", intID, "Transmitter", intTransmitterID)

					fsm.LatestPeerList = p.Peerlist
					
					if intID > intTransmitterID {
						println("Min ID større")
						fsm.StoredOrders = mergeOrders(LatestStatusFromPrimary.Orders, p.Orders)
						fsm.PrimaryID = ID
						fsm.BackupID = p.TransmitterID

					} else if intID < intTransmitterID {
						println("Min ID mindre")
						fsm.PrimaryID = p.TransmitterID
						fsm.BackupID = ""
					}

				} else {

					if p.TransmitterID != ID {
						fsm.PrimaryID = p.TransmitterID
					}
					if p.ReceiverID == ID {
						fsm.BackupID = ID
						isBackup = true
					}

					timeout = time.After(2 * time.Second)
				}  else if p.Version > fsm.Version {
					fmt.Println("Primary version higher. accepting new primary")
					fsm.Version = p.Version
					fsm.PrimaryID = p.TransmitterID
					timeout = time.After(3 * time.Second)

				}

			}
		}

		if fsm.BackupID == ID {

			select {
			case p := <-primaryStatusRX:

				LatestStatusFromPrimary = p
				fsm.StoredOrders = p.Orders
				fsm.IpToIndexMap = p.Map
				fsm.Version = p.Version
				fsm.LatestPeerList = p.Peerlist

				timeout = time.After(2 * time.Second)

			case <-timeout:
				

			}
		}
	}

}
*/

func mergeOrders(orders1 [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool, orders2 [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool) [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool {
	var mergedOrders [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool

	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons-1; j++ {
			for k := 0; k < fsm.MElevators; k++ {
				if orders1[k][i][j] || orders2[k][i][j] {
					mergedOrders[k][i][j] = true
				}
			}
		}
	}
	return mergedOrders
}

func removeFromActivePeers(ID string, peerlist peers.PeerUpdate) peers.PeerUpdate {
	fmt.Print("Id to remove", ID, "Peerlist", peerlist)
	newPeerList := make([]string, 0)
	lostPeers := make([]string, 0)
	for i := 0; i < len(peerlist.Peers); i++ {
		if peerlist.Peers[i] != ID {
			print("Adding", peerlist.Peers[i])
			newPeerList = append(newPeerList, peerlist.Peers[i]) //kopierer over alle noder som ikke skal fjernes i new peer list
		} else {
			lostPeers = append(lostPeers, peerlist.Peers[i])

		}
		for i := 0; i < len(peerlist.Lost); i++ {
			print("Adding to lost", peerlist.Lost[i])
			lostPeers = append(lostPeers, peerlist.Lost[i]) //kopierer over de andre på lost peers
		}

	}
	return peers.PeerUpdate{Peers: newPeerList, Lost: lostPeers, New: peerlist.New}

}
