package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"fmt"
	"time"
)

func Primary(ID string) {

	for {
		if ID == fsm.PrimaryID {
			println("Primary", ID)
			statusTX := make(chan fsm.Status)
			orderTX := make(chan fsm.Order)
			orderRX := make(chan fsm.Order)
			nodeStatusRX := make(chan fsm.SingleElevatorStatus)
			RXFloorReached := make(chan fsm.Order)
			LatestPeerList := peers.PeerUpdate{}
			TXLightUpdates := make(chan fsm.LightUpdate)

			//peerTX := make(chan bool)
			peersRX := make(chan peers.PeerUpdate)

			go peers.Receiver(12055, peersRX)
			go bcast.Transmitter(13055, statusTX)
			go bcast.Transmitter(13056, orderTX)
			go bcast.Receiver(13057, orderRX)
			go bcast.Receiver(13058, RXFloorReached)
			go bcast.Receiver(13059, nodeStatusRX)
			go bcast.Transmitter(13060, TXLightUpdates)

			ticker := time.NewTicker(1 * time.Second)

			for {
				if ID == fsm.PrimaryID {
					if fsm.TakeOverInProgress {
						//do stuff
						distributeOrdersFromLostNode(fsm.PreviousPrimaryID, fsm.StoredOrders, LatestPeerList, orderTX)
						fsm.TakeOverInProgress = false
					}
					select {
					case nodeUpdate := <-nodeStatusRX:

						updateNodeMap(nodeUpdate.ID, nodeUpdate)
					case p := <-peersRX:
						LatestPeerList = p
						fmt.Println(LatestPeerList.Lost)
						if fsm.BackupID == "" && len(p.Peers) > 1 {
							for i := 0; i < len(p.Peers); i++ {
								if p.Peers[i] != ID {
									fsm.BackupID = p.Peers[i]
								}
							}
						}
						if string(p.New) != "" {
							index, exists := getOrAssignIndex(string(p.New))

							fmt.Print("map: ", fsm.IpToIndexMap)

							if exists {
								// Retrieve CAB calls.
								// kanskje vi kan lage en "fake" new order? Eventuelt om vi bør endre single elevator til å ikke være event basert, men heller "while requests in queue"
								//Hvis vi finner at det er lagret cab calls for denne heisen som ikke er gjort her i remote, så trigger vi en ny ordre .
								for i := 0; i < fsm.NFloors; i++ {
									fmt.Print(fsm.StoredOrders[i][2][index])
									if fsm.StoredOrders[i][2][index] {
										print("prevusly stored cab call")
										newOrder := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: 2},
											ID:       ID,
											TargetID: string(p.New),
											Orders:   extractOrder(fsm.StoredOrders, index)}
										fmt.Print("restored cabcalls: ", newOrder.Orders, " for ", newOrder.TargetID)
										orderTX <- newOrder

									}
								}

								println("Retrieving CAB calls")
							}
						}

						for i := 0; i < len(p.Lost); i++ {
							//alle som dør
							print("Lost: ", p.Lost[i])
							fsm.StoredOrders = distributeOrdersFromLostNode(p.Lost[i], fsm.StoredOrders, LatestPeerList, orderTX)

							//hvis backup dør
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
						//sending status to backup

						statusTX <- fsm.Status{TransmitterID: ID, ReceiverID: fsm.BackupID, Orders: fsm.StoredOrders, Version: fsm.Version, Map: fsm.IpToIndexMap}
						//periodic light update to nodes.
						for i := 0; i < len(LatestPeerList.Peers); i++ {
							lightUpdate := fsm.LightUpdate{LightArray: makeLightMatrix(searchMap(i), fsm.StoredOrders), ID: searchMap(i)}
							TXLightUpdates <- lightUpdate

						}
					case a := <-orderRX:
						//Hall assignment

						//Update StoredOrders
						responsibleElevator := AssignRequest(a, LatestPeerList)
						responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)

						fsm.StoredOrders[a.ButtonEvent.Floor][a.ButtonEvent.Button][responsibleElevatorIndex] = true
						//sent to backup in next status update

						newMessage := fsm.Order{ButtonEvent: a.ButtonEvent, ID: ID, TargetID: searchMap(responsibleElevatorIndex), Orders: extractOrder(fsm.StoredOrders, responsibleElevatorIndex)}
						//vi bør kanskje forsikre oss om at backup har lagret dette. Mulig vi må kreve ack fra backup, da vi bruker dette som knappelys garanti.
						orderTX <- newMessage
					case a := <-RXFloorReached:
						if a.ID != "" { //liker ikke dennne her):

							index, _ := getIndex(a.ID)

							fsm.StoredOrders = updateOrders(a.Orders, index)
						}

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
func getIndex(ip string) (int, bool) {
	if index, exists := fsm.IpToIndexMap[ip]; exists {

		return index, true
	} else {
		return -1, false
	}

}
func getOrAssignIndex(ip string) (int, bool) {

	if index, exists := fsm.IpToIndexMap[ip]; exists {

		return index, true
	} else {

		fsm.IpToIndexMap[ip] = len(fsm.IpToIndexMap)
		print("ip ", ip, "not found")

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

func makeLightMatrix(ID string, StoredOrders [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool) [fsm.NFloors][fsm.NButtons]bool {
	var lightMatrix [fsm.NFloors][fsm.NButtons]bool
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons-1; j++ { //we exclude the cab button here
			for k := 0; k < fsm.MElevators; k++ {

				lightMatrix[i][j] = lightMatrix[i][j] || StoredOrders[i][j][k]
			}
		}
	}
	for i := 0; i < fsm.NFloors; i++ {
		nodeIndex, _ := getOrAssignIndex(ID)
		lightMatrix[i][2] = extractOrder(StoredOrders, nodeIndex)[i][2]
	}

	return lightMatrix
}

func distributeOrdersFromLostNode(lostNodeID string, StoredOrders [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool, LatestPeerList peers.PeerUpdate, orderTX chan<- fsm.Order) [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool {
	distributedOrders := StoredOrders
	lostNodeIndex, _ := getIndex(lostNodeID)
	print("lost node index: ", lostNodeID)
	lostOrders := extractOrder(StoredOrders, lostNodeIndex)
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons-1; j++ {
			if lostOrders[i][j] {
				lostOrder := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: elevio.ButtonType(j)}, ID: lostNodeID, TargetID: "", Orders: lostOrders}
				responsibleElevator := AssignRequest(lostOrder, LatestPeerList)
				lostOrder.TargetID = responsibleElevator
				fmt.Print("lost order: ", lostOrder)
				orderTX <- lostOrder
				responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)
				distributedOrders[i][j][responsibleElevatorIndex] = true

			}
		}
	}
	return distributedOrders

}
