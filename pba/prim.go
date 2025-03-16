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

			ticker := time.NewTicker(200 * time.Millisecond)
			prevLightMatrix := [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool{}

			for {
				if ID == fsm.PrimaryID {
					if fsm.TakeOverInProgress {
						//do stuff
						fmt.Println("Takeover in progress", fsm.LatestPeerList)

						fsm.StoredOrders = distributeOrdersFromLostNode(fsm.PreviousPrimaryID, fsm.StoredOrders, orderTX)
						fsm.TakeOverInProgress = false
					}
					select {
					case nodeUpdate := <-nodeStatusRX:

						updateNodeMap(nodeUpdate.ID, nodeUpdate)
					case p := <-peersRX:
						fsm.LatestPeerList = p
						fmt.Println(fsm.LatestPeerList.Lost)
						if fsm.BackupID == "" && len(p.Peers) > 1 {
							for i := 0; i < len(p.Peers); i++ {
								if p.Peers[i] != ID {
									fsm.BackupID = p.Peers[i]
								}
							}
						}
						if string(p.New) != "" {
							index, exists := getOrAssignIndex(string(p.New))

							if exists {
								// Retrieve CAB calls.
								// kanskje vi kan lage en "fake" new order? Eventuelt om vi bør endre single elevator til å ikke være event basert, men heller "while requests in queue"
								//Hvis vi finner at det er lagret cab calls for denne heisen som ikke er gjort her i remote, så trigger vi en ny ordre .
								for i := 0; i < fsm.NFloors; i++ {
									fmt.Print(fsm.StoredOrders[index][i][2])
									if fsm.StoredOrders[index][i][2] {
										print("prevusly stored cab call")
										newOrder := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: 2},
											ID:       ID,
											TargetID: string(p.New),
											Orders:   fsm.StoredOrders[index]}
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
							fsm.StoredOrders = distributeOrdersFromLostNode(p.Lost[i], fsm.StoredOrders, orderTX)

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

						statusTX <- fsm.Status{TransmitterID: ID, ReceiverID: fsm.BackupID, Orders: fsm.StoredOrders, Version: fsm.Version, Map: fsm.IpToIndexMap, Peerlist: fsm.LatestPeerList}
						//periodic light update to nodes.

						//when it is time to send light update:
						// for each node that is active:
						for i := 0; i < len(fsm.LatestPeerList.Peers); i++ {
							//compute the new lightmatrix given the stored orders.
							lightUpdate := makeLightMatrix(fsm.LatestPeerList.Peers[i])
							//problem. denne oppdaterer kun hall light for 1 node av gangen, men denne oppdateringen må gå på alle.

							//if the new lightmatrix is different from the previous lights for the node:
							if lightsDifferent(lightUpdate, prevLightMatrix[i]) {
								TXLightUpdates <- fsm.LightUpdate{LightArray: lightUpdate, ID: fsm.LatestPeerList.Peers[i]}
								//send out the updated lightmatrix to the node.
								prevLightMatrix[i] = lightUpdate //update the previous lightmatrix.
							}
						}

					case a := <-orderRX:
						print("order received from node", a.ID)
						//Hall assignment

						//Update StoredOrders
						responsibleElevator := AssignRequest(a, fsm.LatestPeerList)
						responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)

						fsm.StoredOrders[responsibleElevatorIndex][a.ButtonEvent.Floor][a.ButtonEvent.Button] = true
						//sent to backup in next status update

						newMessage := fsm.Order{ButtonEvent: a.ButtonEvent, ID: ID, TargetID: searchMap(responsibleElevatorIndex), Orders: fsm.StoredOrders[responsibleElevatorIndex]}
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
	print("looking for ip ,", ip)
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

func makeLightMatrix(ID string) [fsm.NFloors][fsm.NButtons]bool {

	lightMatrix := fsm.StoredOrders[fsm.IpToIndexMap[ID]]
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

func distributeOrdersFromLostNode(lostNodeID string, StoredOrders [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool, orderTX chan<- fsm.Order) [fsm.MElevators][fsm.NFloors][fsm.NButtons]bool {
	distributedOrders := StoredOrders
	lostNodeIndex, _ := getIndex(lostNodeID)
	print("lost node index: ", lostNodeID)
	lostOrders := StoredOrders[lostNodeIndex]
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons-1; j++ {
			if lostOrders[i][j] {
				lostOrder := fsm.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: elevio.ButtonType(j)}, ID: lostNodeID, TargetID: "", Orders: lostOrders}
				responsibleElevator := AssignRequest(lostOrder, fsm.LatestPeerList)
				lostOrder.TargetID = responsibleElevator
				fmt.Print("lost order: ", lostOrder)

				distributedOrders[lostNodeIndex][i][j] = false
				fmt.Print("orders of lost node: ", distributedOrders[lostNodeIndex])
				print("responsible elevator: ", responsibleElevator)
				responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)
				distributedOrders[responsibleElevatorIndex][i][j] = true

				orderTX <- lostOrder

			}
		}
	}
	return distributedOrders

}
func lightsDifferent(lightArray1 [fsm.NFloors][fsm.NButtons]bool, lightArray2 [fsm.NFloors][fsm.NButtons]bool) bool {
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons; j++ {
			if lightArray1[i][j] != lightArray2[i][j] {
				return true
			}
		}
	}
	return false
}
