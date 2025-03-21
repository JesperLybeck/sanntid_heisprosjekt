package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/elevio"
	"Sanntid/config"
	"fmt"
	"time"
)

type PrimaryChannels struct {
    OrderTX        chan config.Order
    OrderRX        chan config.Order
    RXFloorReached chan config.Order
	StatusTX       chan config.Status
    NodeStatusRX   chan config.SingleElevatorStatus
	TXLightUpdates chan config.LightUpdate
}

func Primary(ID string, channels PrimaryChannels, peerChannel chan peers.PeerUpdate) {
	for {
		if ID == config.PrimaryID{

			println("Primary", ID)

			go peers.Receiver(12055, peerChannel)

			go bcast.Transmitter(13055, channels.StatusTX)
			go bcast.Transmitter(13056, channels.OrderTX)
			go bcast.Receiver(13057, channels.OrderRX)
			go bcast.Receiver(13058, channels.RXFloorReached)
			go bcast.Receiver(13059, channels.NodeStatusRX)
			go bcast.Transmitter(13060, channels.TXLightUpdates)

			ticker := time.NewTicker(200 * time.Millisecond)
			prevLightMatrix := [config.MElevators][config.NFloors][config.NButtons]bool{}

			for {
				if ID == config.PrimaryID{
					if config.TakeOverInProgress {
						//do stuff
						fmt.Println("Takeover in progress", config.LatestPeerList)

						config.StoredOrders = distributeOrdersFromLostNode(config.PreviousPrimaryID, config.StoredOrders, channels.OrderTX)
						config.TakeOverInProgress = false
					}
					select {
					case nodeUpdate := <-channels.NodeStatusRX:

						updateNodeMap(nodeUpdate.ID, nodeUpdate)

					case p := <-peerChannel:
						config.AloneOnNetwork = false
						config.LatestPeerList = p
						fmt.Println(config.LatestPeerList.Lost)
						if config.BackupID == "" && len(p.Peers) > 1 {
							for i := 0; i < len(p.Peers); i++ {
								if p.Peers[i] != ID {
									config.BackupID = p.Peers[i]
								}
							}
						}
						fmt.Println(p.Peers)
	
						if string(p.New) != "" {
							index, exists := getOrAssignIndex(string(p.New))

							if exists {
								// Retrieve CAB calls.
								// kanskje vi kan lage en "fake" new order? Eventuelt om vi bør endre single elevator til å ikke være event basert, men heller "while requests in queue"
								//Hvis vi finner at det er lagret cab calls for denne heisen som ikke er gjort her i remote, så trigger vi en ny ordre .
								for i := 0; i < config.NFloors; i++ {
									fmt.Print(config.StoredOrders[index][i][2])
									if config.StoredOrders[index][i][2] {
										print("prevusly stored cab call")
										newOrder := config.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: 2},
											ID:       ID,
											TargetID: string(p.New),
											Orders:   config.StoredOrders[index]}
										fmt.Print("restored cabcalls: ", newOrder.Orders, " for ", newOrder.TargetID)
										channels.OrderTX <- newOrder

									}
								}

								println("Retrieving CAB calls")
							}
						}

						for i := 0; i < len(p.Lost); i++ {
							//alle som dør
							print("Lost: ", p.Lost[i])
							config.StoredOrders = distributeOrdersFromLostNode(p.Lost[i], config.StoredOrders, channels.OrderTX)

							//hvis backup dør
							if p.Lost[i] == config.BackupID {
								println("Backup lost")
								for j := 0; j < len(p.Peers); j++ {
									if p.Peers[j] != config.PrimaryID{
										config.BackupID = p.Peers[j]
									} else {
										config.BackupID = ""
									}
								}
							}
						}

					case <-ticker.C:
						//sending status to backup
						fmt.Println("I am Primary")
						channels.StatusTX <- config.Status{TransmitterID: ID, ReceiverID: config.BackupID, Orders: config.StoredOrders, Version: config.Version, Map: config.IpToIndexMap, Peerlist: config.LatestPeerList}
						//periodic light update to nodes.

						//when it is time to send light update:
						// for each node that is active:
						for i := 0; i < len(config.LatestPeerList.Peers); i++ {
							//compute the new lightmatrix given the stored orders.
							lightUpdate := makeLightMatrix(config.LatestPeerList.Peers[i])
							//problem. denne oppdaterer kun hall light for 1 node av gangen, men denne oppdateringen må gå på alle.

							//if the new lightmatrix is different from the previous lights for the node:
							if lightsDifferent(lightUpdate, prevLightMatrix[i]) {
								channels.TXLightUpdates <- config.LightUpdate{LightArray: lightUpdate, ID: config.LatestPeerList.Peers[i]}
								//send out the updated lightmatrix to the node.
								prevLightMatrix[i] = lightUpdate //update the previous lightmatrix.
							}
						}

					case a := <-channels.OrderRX:
						print("order received from node", a.ID)
						//Hall assignment

						//Update StoredOrders
						responsibleElevator := AssignRequest(a, config.LatestPeerList)
						responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator)

						config.StoredOrders[responsibleElevatorIndex][a.ButtonEvent.Floor][a.ButtonEvent.Button] = true
						//sent to backup in next status update

						newMessage := config.Order{ButtonEvent: a.ButtonEvent, ID: ID, TargetID: searchMap(responsibleElevatorIndex), Orders: config.StoredOrders[responsibleElevatorIndex]}
						//vi bør kanskje forsikre oss om at backup har lagret dette. Mulig vi må kreve ack fra backup, da vi bruker dette som knappelys garanti.
						channels.OrderTX <- newMessage

					case a := <-channels.RXFloorReached:
						if a.ID != "" { //liker ikke dennne her):

							index, _ := getIndex(a.ID)

							config.StoredOrders = updateOrders(a.Orders, index)

						}

					}

				}
			}
		}
	}
}

func updateOrders(ordersFromNode [config.NFloors][config.NButtons]bool, elevator int) [config.MElevators][config.NFloors][config.NButtons]bool {

	newStoredOrders := config.StoredOrders

	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons; j++ {
			newStoredOrders[elevator][i][j] = ordersFromNode[i][j]
		}
	}
	// Copy existing orders
	/*for i := range config.StoredOrders {
		for j := range config.StoredOrders[i] {
			copy(newStoredOrders[i][j][:], config.StoredOrders[i][j][:])
		}
	}
	newStoredOrders[elevator] = ordersFromNode*/

	return newStoredOrders
}
func getIndex(ip string) (int, bool) {
	print("looking for ip ,", ip)
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
		print("ip ", ip, "not found")

		return config.IpToIndexMap[ip], false
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

func updateNodeMap(ID string, status config.SingleElevatorStatus) {
	if _, exists := config.NodeStatusMap[ID]; exists {
		config.NodeStatusMap[ID] = status
	} else {
		config.NodeStatusMap[ID] = status
	}
}

func makeLightMatrix(ID string) [config.NFloors][config.NButtons]bool {

	lightMatrix := config.StoredOrders[config.IpToIndexMap[ID]]
	for floor := 0; floor < config.NFloors; floor++ {
		for button := 0; button < config.NButtons-1; button++ {
			for elev := 0; elev < config.MElevators; elev++ {
				if config.StoredOrders[elev][floor][button] {
					lightMatrix[floor][button] = true
				}
			}
		}
	}

	for floor := 0; floor < config.NFloors; floor++ {
		lightMatrix[floor][2] = config.StoredOrders[config.IpToIndexMap[ID]][floor][2] //setter cab lights inidividuelt
	}

	return lightMatrix
}

func distributeOrdersFromLostNode(lostNodeID string, StoredOrders [config.MElevators][config.NFloors][config.NButtons]bool, orderTX chan<- config.Order) [config.MElevators][config.NFloors][config.NButtons]bool {
	distributedOrders := StoredOrders
	lostNodeIndex, _ := getIndex(lostNodeID)
	print("lost node index: ", lostNodeID)
	lostOrders := StoredOrders[lostNodeIndex]
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons-1; j++ {
			if lostOrders[i][j] {
				lostOrder := config.Order{ButtonEvent: elevio.ButtonEvent{Floor: i, Button: elevio.ButtonType(j)}, ID: lostNodeID, TargetID: "", Orders: lostOrders}
				responsibleElevator := AssignRequest(lostOrder, config.LatestPeerList)
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
func lightsDifferent(lightArray1 [config.NFloors][config.NButtons]bool, lightArray2 [config.NFloors][config.NButtons]bool) bool {
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons; j++ {
			if lightArray1[i][j] != lightArray2[i][j] {
				return true
			}
		}
	}
	return false
}
