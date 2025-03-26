package pba

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/peers"
	"fmt"
	"time"
)

var OrderNumber int = 1

func Primary(ID string, primaryElection <-chan network.Election) {

	for {
		if ID == PrimaryID {

			//TODO, PASS CHANNELS I GOROUTINES

			statusTX := make(chan network.Status)
			orderTX := make(chan network.Order)
			requestRX := make(chan network.Request)
			nodeStatusRX := make(chan network.SingleElevatorStatus)
			RXFloorReached := make(chan network.Request)

			TXLightUpdates := make(chan network.LightUpdate)

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
			lightUpdateTicker := time.NewTicker(50 * time.Millisecond)

			var lastMessagesMap = make(map[string]int)

			for {
				if ID == PrimaryID {
					if TakeOverInProgress {
						//do stuff

						StoredOrders = distributeOrdersFromLostNode(PreviousPrimaryID, StoredOrders, orderTX, nodeStatusRX, requestRX)
						TakeOverInProgress = false
					}
					select {
					case p := <-primaryElection:
						fmt.Print(p)

						PrimaryID = p.PrimaryID
						BackupID = p.BackupID

					case nodeUpdate := <-nodeStatusRX:

						updateNodeMap(nodeUpdate.ID, nodeUpdate)
					case p := <-peersRX:
						print("new peer update")
						AloneOnNetwork = false
						fmt.Println("Peerupdate in prim, change of LatestPeerList")
						LatestPeerList = p

						if BackupID == "" && len(p.Peers) > 1 {
							for i := 0; i < len(p.Peers); i++ {
								if p.Peers[i] != ID {
									BackupID = p.Peers[i]
								}
							}
						}

						if p.New != "" {
							index, exists := getOrAssignIndex(p.New)

							if exists {
								// Retrieve CAB calls.
								// kanskje vi kan lage en "fake" new order? Eventuelt om vi bør endre single elevator til å ikke være event basert, men heller "while requests in queue"
								//Hvis vi finner at det er lagret cab calls for denne heisen som ikke er gjort her i remote, så trigger vi en ny ordre .
								for i := 0; i < config.NFloors; i++ {

									if StoredOrders[index][i][elevator.BT_Cab] {

										newOrder := network.Order{ButtonEvent: elevator.ButtonEvent{Floor: i, Button: elevator.BT_Cab},
											ResponisbleElevator: p.New,
											OrderID:             OrderNumber,
										}
										print("new order from restore cab calls")
										print("----------searchmapindex:-------->", searchMap(index), "<------")
										go network.SendOrder(orderTX, nodeStatusRX, newOrder, searchMap(index), OrderNumber, requestRX, NodeStatusMap)

										OrderNumber++

									}
								}

							}
						}
						fmt.Print("lost node", p.Lost)
						for i := 0; i < len(p.Lost); i++ {
							//alle som dør

							StoredOrders = distributeOrdersFromLostNode(p.Lost[i], StoredOrders, orderTX, nodeStatusRX, requestRX)

							//hvis backup dør
							if p.Lost[i] == BackupID {

								for j := 0; j < len(p.Peers); j++ {
									if p.Peers[j] != PrimaryID {
										BackupID = p.Peers[j]
									} else {
										BackupID = ""
									}
								}
							}
						}

					case <-ticker.C:
						//sending status to backup

						statusTX <- network.Status{TransmitterID: ID, ReceiverID: BackupID, Orders: StoredOrders, Map: config.IpToIndexMap}
						//periodic light update to nodes.

						//when it is time to send light update:
						// for each node that is active:

					case <-lightUpdateTicker.C:

						for i := 0; i < len(config.IpToIndexMap); i++ {
							//compute the new lightmatrix given the stored orders.
							lightUpdate := makeLightMatrix(searchMap(i))
							//problem. denne oppdaterer kun hall light for 1 node av gangen, men denne oppdateringen må gå på alle.

							//if the new lightmatrix is different from the previous lights for the node:

							TXLightUpdates <- network.LightUpdate{LightArray: lightUpdate, ID: searchMap(i)}
							//send out the updated lightmatrix to the node.
						}

					case a := <-requestRX:

						//Hall assignment
						/*if hallCallAssigned(a) {
							continue
						}*/
						//Update StoredOrders

						lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, lastMessagesMap)
						if lastMessageNumber == a.RequestID {

							continue
						}
						fmt.Print("LastMessageNumber-->", lastMessageNumber, "--RequestID-->", a.RequestID, "---")

						order := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: a.ID, OrderID: OrderNumber}

						responsibleElevator := AssignOrder(order, LatestPeerList, NodeStatusMap)
						print("responsible elevator", responsibleElevator)
						order.ResponisbleElevator = responsibleElevator

						responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)

						StoredOrders[responsibleElevatorIndex][a.ButtonEvent.Floor][a.ButtonEvent.Button] = true
						//sent to backup in next status update

						newMessage := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: responsibleElevator, OrderID: OrderNumber}

						//vi bør kanskje forsikre oss om at backup har lagret dette. Mulig vi må kreve ack fra backup, da vi bruker dette som knappelys garanti.

						go network.SendOrder(orderTX, nodeStatusRX, newMessage, ID, OrderNumber, requestRX, NodeStatusMap)
						print("new order from request")
						OrderNumber++
						lastMessagesMap[a.ID] = a.RequestID

					case a := <-RXFloorReached:

						fmt.Println(a)

						lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, lastMessagesMap)
						if lastMessageNumber == a.RequestID {

							continue
						}
						if a.ID != "" { //liker ikke dennne her):

							index, _ := getOrAssignIndex(a.ID) //kan bli -1 hvis vi ikke er i mappet.

							StoredOrders = updateOrders(a.Orders, index)

							lastMessagesMap[a.ID] = a.RequestID
						}

					}

				}
			}
		}
	}
}

func updateOrders(ordersFromNode [config.NFloors][config.NButtons]bool, elevator int) [config.MElevators][config.NFloors][config.NButtons]bool {
	if elevator == -1 {

	}
	newStoredOrders := StoredOrders

	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons; j++ {

			newStoredOrders[elevator][i][j] = ordersFromNode[i][j]
		}
	}

	// Copy existing orders
	/*for i := range StoredOrders {
		for j := range StoredOrders[i] {
			copy(newStoredOrders[i][j][:], StoredOrders[i][j][:])
		}
	}
	newStoredOrders[elevator] = ordersFromNode*/

	return newStoredOrders
}
func getIndex(ip string) (int, bool) {

	if index, exists := config.IpToIndexMap[ip]; exists {

		return index, true
	} else {
		return -1, false
	}

}
func getOrAssignIndex(ip string) (int, bool) {

	if index, exists := config.IpToIndexMap[ip]; exists {

		return index, true
	} else {

		config.IpToIndexMap[ip] = len(config.IpToIndexMap)

		return config.IpToIndexMap[ip], false
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
	for key, value := range config.IpToIndexMap {
		if value == index {
			return key
		}
	}

	return ""
}

func updateNodeMap(ID string, status network.SingleElevatorStatus) {
	if _, exists := NodeStatusMap[ID]; exists {
		NodeStatusMap[ID] = status
	} else {
		NodeStatusMap[ID] = status
	}
}

func makeLightMatrix(ID string) [config.NFloors][config.NButtons]bool {

	lightMatrix := [config.NFloors][config.NButtons]bool{}

	for floor := 0; floor < config.NFloors; floor++ {
		for button := 0; button < config.NButtons-1; button++ {
			for elev := 0; elev < config.MElevators; elev++ {
				if StoredOrders[elev][floor][button] {
					lightMatrix[floor][button] = true
				}

			}
		}
	}

	for floor := 0; floor < config.NFloors; floor++ {
		lightMatrix[floor][2] = StoredOrders[config.IpToIndexMap[ID]][floor][2] //setter cab lights inidividuelt
	}

	return lightMatrix
}

func distributeOrdersFromLostNode(lostNodeID string, StoredOrders [config.MElevators][config.NFloors][config.NButtons]bool, orderTX chan<- network.Order, ackChan <-chan network.SingleElevatorStatus, resendChan chan network.Request) [config.MElevators][config.NFloors][config.NButtons]bool {
	distributedOrders := StoredOrders
	lostNodeIndex, _ := getOrAssignIndex(lostNodeID)

	lostOrders := StoredOrders[lostNodeIndex]
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons-1; j++ {
			if lostOrders[i][j] {
				lostOrder := network.Order{ButtonEvent: elevator.ButtonEvent{Floor: i, Button: elevator.ButtonType(j)}, ResponisbleElevator: "", OrderID: OrderNumber}
				responsibleElevator := AssignOrder(lostOrder, LatestPeerList, NodeStatusMap)
				lostOrder.ResponisbleElevator = responsibleElevator

				distributedOrders[lostNodeIndex][i][j] = false

				responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)
				distributedOrders[responsibleElevatorIndex][i][j] = true
				print("new order from lost node")
				go network.SendOrder(orderTX, ackChan, lostOrder, responsibleElevator, OrderNumber, resendChan, NodeStatusMap)
				OrderNumber++

			}
		}
	}
	return distributedOrders

}
