package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/fsm"
	"time"
)

func Primary(ID string) {

	for {
		if ID == fsm.PrimaryID {
			statusTX := make(chan fsm.Status)
			orderTX := make(chan fsm.Order)
			orderRX := make(chan fsm.Order)
			nodeStatusRX := make(chan fsm.SingleElevatorStatus)
			RXFloorReached := make(chan fsm.Order)
			latestPeerList := peers.PeerUpdate{}

			//peerTX := make(chan bool)
			peersRX := make(chan peers.PeerUpdate)

			go peers.Receiver(12055, peersRX)
			go bcast.Transmitter(13055, statusTX)
			go bcast.Transmitter(13056, orderTX)
			go bcast.Receiver(13057, orderRX)
			go bcast.Receiver(13058, RXFloorReached)
			go bcast.Receiver(13059, nodeStatusRX)

			ticker := time.NewTicker(2 * time.Second)

			for {
				if ID == fsm.PrimaryID {
					select {
					case nodeUpdate := <-nodeStatusRX:
						//Update stored orders

						updateNodeMap(nodeUpdate.ID, nodeUpdate)
					case p := <-peersRX:
						latestPeerList = p
						if fsm.BackupID == "" && len(p.Peers) > 1 {
							for i := 0; i < len(p.Peers); i++ {
								if p.Peers[i] != ID {
									fsm.BackupID = p.Peers[i]
								}
							}
						}

						_, exists := getOrAssignIndex(string(p.New))

						if exists {
							// Retrieve CAB calls.
							// kanskje vi kan lage en "fake" new order? Eventuelt om vi bør endre single elevator til å ikke være event basert, men heller "while requests in queue"

							println("Retrieving CAB calls")
						}

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
						responsibleElevator := AssignRequest(a, latestPeerList)
						responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)
						fsm.StoredOrders[a.ButtonEvent.Floor][a.ButtonEvent.Button][responsibleElevatorIndex] = true
						//sent to backup in next status update

						newMessage := fsm.Order{ButtonEvent: a.ButtonEvent, ID: ID, TargetID: searchMap(responsibleElevatorIndex), Orders: extractOrder(fsm.StoredOrders, responsibleElevatorIndex)}
						//vi bør kanskje forsikre oss om at backup har lagret dette. Mulig vi må kreve ack fra backup, da vi bruker dette som knappelys garanti.
						orderTX <- newMessage
					case a := <-RXFloorReached:
						index, _ := getOrAssignIndex(string(a.ID))
						fsm.StoredOrders = updateOrders(a.Orders, index)
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

func updateOrders(StoredOrders [fsm.NFloors][fsm.NButtons]bool, elevator int) [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool {
	var orders [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons; j++ {
			orders[i][j][elevator] = StoredOrders[i][j]
		}
	}
	return orders
}

func getOrAssignIndex(ip string) (int, bool) {

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

func updateNodeMap(ID string, status fsm.SingleElevatorStatus) {
	if _, exists := fsm.NodeStatusMap[ID]; exists {
		fsm.NodeStatusMap[ID] = status
	} else {
		fsm.NodeStatusMap[ID] = status
	}
}
