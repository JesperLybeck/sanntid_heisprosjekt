package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/fsm"
	"fmt"
	"time"
)

func Primary(ID string) {

	for {
		if ID == fsm.PrimaryID {
			statusTX := make(chan fsm.Status)
			orderTX := make(chan fsm.Order)
			orderRX := make(chan fsm.Order)

			//peerTX := make(chan bool)
			peersRX := make(chan peers.PeerUpdate)

			go peers.Receiver(12055, peersRX)
			go bcast.Transmitter(13055, statusTX)
			go bcast.Transmitter(13056, orderTX)
			go bcast.Receiver(13057, orderRX)

			ticker := time.NewTicker(2 * time.Second)

			for {
				if ID == fsm.PrimaryID {
					select {
					case p := <-peersRX:
						if fsm.BackupID == "" && len(p.Peers) > 1 {
							for i := 0; i < len(p.Peers); i++ {
								if p.Peers[i] != ID {
									fsm.BackupID = p.Peers[i]
								}
							}
						}
						
						index, exists := getOrAssignIndex(string(p.New))
						println("Index",index, "IP", p.New)
						println("Ip searched in map ", searchMap(index))
						if exists {
							// Retrieve CAB calls.
						}
						println("Retrieving CAB calls")
						fmt.Println("Peer update", p.Peers)
						fmt.Println("New", p.New)
						fmt.Println("Lost", p.Lost)
	
						
						// LAG EN MAPPING MELLOM HEISINDEKS OG ID

						for i := 0; i < len(p.Lost); i++ {
							if p.Lost[i] == fsm.BackupID {
								println("Backup lost")
								for j := 0; j < len(p.Peers); j++ {
									if p.Peers[j] != fsm.PrimaryID {
										fsm.BackupID = p.Peers[j]
									} else {
										fsm.BackupID = ""
									}
								}
							}
						}

					case <-ticker.C:
						statusTX <- fsm.Status{TransmitterID: ID, ReceiverID: fsm.BackupID, Orders: fsm.StoredOrders, Version: fsm.Version}

					
					case a := <-orderRX:
						//Hall assignment
						
						//Update StoredOrders
						var responsibleElevator int
						fsm.StoredOrders,responsibleElevator = AssignRequest(a, fsm.StoredOrders)
						//sent to backup in next status update
						println("Responsible elevator", responsibleElevator)
						newMessage := fsm.Order{ButtonEvent: a.ButtonEvent, ID: ID, TargetID: searchMap(responsibleElevator), Orders: extractOrder(fsm.StoredOrders, responsibleElevator)}

						orderTX <- newMessage
					}
				}
			}
		}
	}
}

func extractOrder(StoredOrders [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool, elevator int) [fsm.NFloors][fsm.NButtons]bool {
	var orders [fsm.NFloors][fsm.NButtons]bool
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons; j++ {
			orders[i][j] = StoredOrders[i][j][elevator]
		}
	}
	return orders
}

func getOrAssignIndex(ip string) (int, bool){
	if index, exists := fsm.IpToIndexMap[ip]; exists {
		return index, true
	} else {
		fsm.IpToIndexMap[ip] = len(fsm.IpToIndexMap)
		return fsm.IpToIndexMap[ip], false
	}
}
func searchMap(index int) string {
	for key, value := range fsm.IpToIndexMap {
		if value == index {
			return key
		}
	}
	return ""
}
