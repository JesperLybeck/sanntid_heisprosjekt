package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/config"
	"fmt"
	"strconv"
	"time"
)

var LatestStatusFromPrimary config.Status



func StatusReciever(ID string, primaryStatusRX chan config.Status) {
	go bcast.Receiver(13055, primaryStatusRX)
	for {
		if config.BackupID != ID {
			select {
			case p := <-primaryStatusRX:

				if config.PrimaryID == ID && p.TransmitterID != ID {
					intID, _ := strconv.Atoi(ID[len(ID)-2:])
					intTransmitterID, _ := strconv.Atoi(p.TransmitterID[len(ID)-2:])
					//Her mottar en primary melding fra en annen primary
					print("MyID", intID, "Transmitter", intTransmitterID)
					config.LatestPeerList = p.Peerlist
					if intID > intTransmitterID {
						println("Min ID større")
						config.StoredOrders = mergeOrders(LatestStatusFromPrimary.Orders, p.Orders)
						config.PrimaryID = ID
						config.BackupID = p.TransmitterID

					} else if intID < intTransmitterID {
						println("Min ID mindre")
						config.PrimaryID = p.TransmitterID
						config.BackupID = ""
					}

				} else {

					if p.TransmitterID != ID {
						config.PrimaryID = p.TransmitterID
						config.BackupID = p.ReceiverID
					}
				}
			}
		}
	}
}
func Backup(ID string, primaryStatusRX chan config.Status) {
	go bcast.Receiver(13055, primaryStatusRX)
	timeout := time.After(3 * time.Second)

	for {

		if config.BackupID == ID {
			select {
			case p := <-primaryStatusRX:
				fmt.Println("I am Backup")
				LatestStatusFromPrimary = p
				config.StoredOrders = p.Orders
				config.IpToIndexMap = p.Map
				config.Version = p.Version
				config.LatestPeerList = p.Peerlist
				timeout = time.After(3 * time.Second)

			case <-timeout:
				fmt.Println("Primary timed out")
				config.LatestPeerList = removeFromActivePeers(config.PrimaryID, config.LatestPeerList)
				fmt.Println("New peerlist", config.LatestPeerList)
				config.Version++
				config.PreviousPrimaryID = config.PrimaryID
				config.PrimaryID = ID
				config.BackupID = ""
				config.TakeOverInProgress = true

			}
		}
	}

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
