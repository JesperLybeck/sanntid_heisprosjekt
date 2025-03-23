package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/fsm"
	"fmt"
	"time"
)

var OrderNumber int = 1

func Primary(ID string) {

	for {
		if ID == fsm.PrimaryID {

			//TODO, PASS CHANNELS I GOROUTINES

			statusTX := make(chan fsm.Status)
			orderTX := make(chan fsm.Order)
			requestRX := make(chan fsm.Request)
			nodeStatusRX := make(chan fsm.SingleElevatorStatus)
			RXFloorReached := make(chan fsm.Request)

			TXLightUpdates := make(chan fsm.LightUpdate)

			//peerTX := make(chan bool)
			peersRX := make(chan peers.PeerUpdate)

			go peers.Receiver(12055, peersRX)
			go bcast.Transmitter(13055, statusTX)
			go bcast.Transmitter(13056, orderTX)
			go bcast.Receiver(13057, requestRX)
			go bcast.Receiver(13058, RXFloorReached)
			go bcast.Receiver(13059, nodeStatusRX)
			go bcast.Transmitter(13060, TXLightUpdates)

			ticker := time.NewTicker(30 * time.Millisecond)

			for {
				if ID == fsm.PrimaryID {
					if fsm.TakeOverInProgress {
						//do stuff

						fsm.StoredOrders = distributeOrdersFromLostNode(fsm.PreviousPrimaryID, fsm.StoredOrders, orderTX, nodeStatusRX)
						fsm.TakeOverInProgress = false
					}
					select {
					case nodeUpdate := <-nodeStatusRX:

						updateNodeMap(nodeUpdate.ID, nodeUpdate)
					case p := <-peersRX:
						fsm.AloneOnNetwork = false
						fsm.LatestPeerList = p

						if fsm.BackupID == "" && len(p.Peers) > 1 {
							for i := 0; i < len(p.Peers); i++ {
								if p.Peers[i] != ID {
									fsm.BackupID = p.Peers[i]
								}
							}
						}

						if string(p.New) != "" {
							index, exists := getOrAssignIndex(string(p.New), fsm.IpToIndexMap)

							if exists {
								// Retrieve CAB calls.
								// kanskje vi kan lage en "fake" new order? Eventuelt om vi bør endre single elevator til å ikke være event basert, men heller "while requests in queue"
								//Hvis vi finner at det er lagret cab calls for denne heisen som ikke er gjort her i remote, så trigger vi en ny ordre .
								for i := 0; i < fsm.NFloors; i++ {

									if fsm.StoredOrders[index][i][2] {

										newOrder := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: elevio.BT_Cab},
											ResponisbleElevator: string(p.New),
											OrderID:             OrderNumber,
										}
										print("new order from restore cab calls")
										go fsm.SendOrder(orderTX, nodeStatusRX, newOrder, ID, OrderNumber)
										OrderNumber++

									}
								}

							}
						}
						fmt.Print("lost node", p.Lost)
						for i := 0; i < len(p.Lost); i++ {
							//alle som dør

							//fsm.StoredOrders = distributeOrdersFromLostNode(p.Lost[i], fsm.StoredOrders, orderTX, nodeStatusRX)

							//hvis backup dør
							if p.Lost[i] == fsm.BackupID {

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

						statusTX <- fsm.Status{TransmitterID: ID, ReceiverID: fsm.BackupID, Orders: fsm.StoredOrders, Version: fsm.Version, Map: fsm.IpToIndexMap, Peerlist: fsm.LatestPeerList}
						//periodic light update to nodes.

						//when it is time to send light update:
						// for each node that is active:
						for i := 0; i < len(fsm.LatestPeerList.Peers); i++ {
							//compute the new lightmatrix given the stored orders.
							lightUpdate := makeLightMatrix(fsm.LatestPeerList.Peers[i])
							//problem. denne oppdaterer kun hall light for 1 node av gangen, men denne oppdateringen må gå på alle.

							//if the new lightmatrix is different from the previous lights for the node:

							TXLightUpdates <- fsm.LightUpdate{LightArray: lightUpdate, ID: fsm.LatestPeerList.Peers[i]}
							//send out the updated lightmatrix to the node.

						}

					case a := <-requestRX:

						//Hall assignment
						/*if hallCallAssigned(a) {
							continue
						}*/
						//Update StoredOrders

						lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, fsm.LastMessagesMap)
						if lastMessageNumber == a.RequestID {

							continue
						}

						order := fsm.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: a.ID, OrderID: OrderNumber}

						responsibleElevator := AssignOrder(order, fsm.LatestPeerList)
						order.ResponisbleElevator = responsibleElevator

						responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator, fsm.IpToIndexMap)

						fsm.StoredOrders[responsibleElevatorIndex][a.ButtonEvent.Floor][a.ButtonEvent.Button] = true
						//sent to backup in next status update

						newMessage := fsm.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: responsibleElevator, OrderID: OrderNumber}

						//vi bør kanskje forsikre oss om at backup har lagret dette. Mulig vi må kreve ack fra backup, da vi bruker dette som knappelys garanti.

						go fsm.SendOrder(orderTX, nodeStatusRX, newMessage, ID, OrderNumber)
						print("new order from request")
						OrderNumber++
						fsm.LastMessagesMap[a.ID] = a.RequestID

					case a := <-RXFloorReached:

						lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, fsm.LastMessagesMap)
						if lastMessageNumber == a.RequestID {

							continue
						}
						if a.ID != "" { //liker ikke dennne her):

							index, _ := getIndex(a.ID)

							fsm.StoredOrders = updateOrders(a.Orders, index)

							lightUpdate := makeLightMatrix(a.ID)

							TXLightUpdates <- fsm.LightUpdate{LightArray: lightUpdate, ID: a.ID}
							fsm.LastMessagesMap[a.ID] = a.RequestID
						}

					}

				}
			}
		}
	}
}

func updateOrders(ordersFromNode [fsm.NFloors][fsm.NButtons]bool, elevator int) [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool {

	newStoredOrders := fsm.StoredOrders

	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons; j++ {

			newStoredOrders[elevator][i][j] = ordersFromNode[i][j]
		}
	}

	// Copy existing orders
	/*for i := range fsm.StoredOrders {
		for j := range fsm.StoredOrders[i] {
			copy(newStoredOrders[i][j][:], fsm.StoredOrders[i][j][:])
		}
	}
	newStoredOrders[elevator] = ordersFromNode*/

	return newStoredOrders
}
func getIndex(ip string) (int, bool) {

	if index, exists := fsm.IpToIndexMap[ip]; exists {

		return index, true
	} else {
		return -1, false
	}

}
func getOrAssignIndex(ip string, IDMap map[string]int) (int, bool) {

	if index, exists := IDMap[ip]; exists {

		return index, true
	} else {

		IDMap[ip] = len(fsm.IpToIndexMap)

		return IDMap[ip], false
	}
}
func getOrAssignMessageNumber(ip string, IDMap map[string]int) (int, bool) {

	if index, exists := IDMap[ip]; exists {

		return index, true
	} else {

		IDMap[ip] = 0

		return IDMap[ip], false
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

func makeLightMatrix(ID string) [fsm.NFloors][fsm.NButtons]bool {

	lightMatrix := [fsm.NFloors][fsm.NButtons]bool{}

	for floor := 0; floor < fsm.NFloors; floor++ {
		for button := 0; button < fsm.NButtons-1; button++ {
			for elev := 0; elev < fsm.MElevators; elev++ {
				if fsm.StoredOrders[elev][floor][button] {
					lightMatrix[floor][button] = true
				}

			}
		}
	}

	for floor := 0; floor < fsm.NFloors; floor++ {
		lightMatrix[floor][2] = fsm.StoredOrders[fsm.IpToIndexMap[ID]][floor][2] //setter cab lights inidividuelt
	}

	return lightMatrix
}

func distributeOrdersFromLostNode(lostNodeID string, StoredOrders [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool, orderTX chan<- fsm.Order, ackChan <-chan fsm.SingleElevatorStatus) [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool {
	distributedOrders := StoredOrders
	lostNodeIndex, _ := getIndex(lostNodeID)

	lostOrders := StoredOrders[lostNodeIndex]
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons-1; j++ {
			if lostOrders[i][j] {
				lostOrder := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: elevio.ButtonType(j)}, ResponisbleElevator: "", OrderID: OrderNumber}
				responsibleElevator := AssignOrder(lostOrder, fsm.LatestPeerList)
				lostOrder.ResponisbleElevator = responsibleElevator

				distributedOrders[lostNodeIndex][i][j] = false

				responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator, fsm.IpToIndexMap)
				distributedOrders[responsibleElevatorIndex][i][j] = true
				print("new order from lost node")
				fsm.SendOrder(orderTX, ackChan, lostOrder, responsibleElevator, OrderNumber)
				OrderNumber++

			}
		}
	}
	return distributedOrders

}
