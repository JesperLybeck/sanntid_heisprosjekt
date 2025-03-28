package pba

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"Sanntid/networkDriver/peers"
	"time"
)

var OrderNumber int = 1

func Primary(id string, primaryElection <-chan network.Election, initialState network.Takeover, done chan<- bool) {
	statusTX := make(chan network.Status)
	orderTX := make(chan network.Order)
	requestRX := make(chan network.Request)
	nodeStatusRX := make(chan network.SingleElevatorStatus)
	RXFloorReached := make(chan network.Request)
	TXLightUpdates := make(chan network.LightUpdate)
	peersRX := make(chan peers.PeerUpdate)

	var storedOrders = initialState.StoredOrders
	var nodeStatusMap = initialState.NodeMap
	var previousprimaryID = initialState.PreviousPrimaryID
	var takeOverInProgress = initialState.TakeOverInProgress
	var latestPeerList = initialState.Peerlist
	var lastMessagesMap = make(map[string]int)

	go peers.Receiver(12055, peersRX)
	go bcast.Transmitter(13055, statusTX)
	go bcast.Transmitter(13056, orderTX)
	go bcast.Receiver(13057, requestRX)
	go bcast.Receiver(13058, RXFloorReached)
	go bcast.Receiver(13059, nodeStatusRX)
	go bcast.Transmitter(13060, TXLightUpdates)

	ticker := time.NewTicker(50 * time.Millisecond)
	lightUpdateTicker := time.NewTicker(50 * time.Millisecond)

	if takeOverInProgress {
		var lostOrders []network.Order
		storedOrders, lostOrders = distributeOrdersFromLostNode(previousprimaryID, storedOrders, config.IDToIndexMap, nodeStatusMap, latestPeerList)
		for order := 0; order < len(lostOrders); order++ {
			go network.SendOrder(orderTX, nodeStatusRX, lostOrders[order], previousprimaryID, OrderNumber, requestRX, nodeStatusMap)
			OrderNumber++
		}
		takeOverInProgress = false
	}

	for {
		select {
		case nodeUpdate := <-nodeStatusRX:
			nodeStatusMap = UpdateNodeMap(nodeUpdate.ID, nodeUpdate, nodeStatusMap)

		case p := <-primaryElection:
			if id != p.PrimaryID {
				done <- true
				return
			}

		case p := <-peersRX:
			latestPeerList = p
			if p.New != "" {
				index, exists := getOrAssignIndex(p.New, config.IDToIndexMap)
				if exists {
					for i := 0; i < config.NFloors; i++ {
						if storedOrders[index][i][elevator.BT_Cab] {
							newOrder := network.Order{ButtonEvent: elevator.ButtonEvent{Floor: i, Button: elevator.BT_Cab},
								ResponisbleElevator: p.New,
								OrderID:             OrderNumber,
							}
							go network.SendOrder(orderTX, nodeStatusRX, newOrder, searchMap(index, config.IDToIndexMap), OrderNumber, requestRX, nodeStatusMap)
							OrderNumber++
						}
					}
				}
			}
			for i := 0; i < len(p.Lost); i++ {
				var lostOrders []network.Order
				storedOrders, lostOrders = distributeOrdersFromLostNode(p.Lost[i], storedOrders, config.IDToIndexMap, nodeStatusMap, latestPeerList)
				for order := 0; order < len(lostOrders); order++ {
					go network.SendOrder(orderTX, nodeStatusRX, lostOrders[order], id, OrderNumber, requestRX, nodeStatusMap)
					OrderNumber++
				}
			}

		case <-ticker.C:
			statusTX <- network.Status{
				TransmitterID:      id,
				Orders:             storedOrders,
				StatusID:           1,
				AloneOnNetwork:     false,
				TakeOverInProgress: false,
			}

		case <-lightUpdateTicker.C:
			for i := 0; i < len(config.IDToIndexMap); i++ {
				lightUpdate := makeLightMatrix(searchMap(i, config.IDToIndexMap), storedOrders, config.IDToIndexMap)
				TXLightUpdates <- network.LightUpdate{LightArray: lightUpdate, ID: searchMap(i, config.IDToIndexMap)}
			}

		case a := <-requestRX:
			lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, lastMessagesMap)
			if lastMessageNumber == a.RequestID {
				continue
			}
			order := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: a.ID, OrderID: OrderNumber}
			responsibleElevator := AssignOrder(order, latestPeerList, nodeStatusMap)
			order.ResponisbleElevator = responsibleElevator
			responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator, config.IDToIndexMap)
			storedOrders[responsibleElevatorIndex][a.ButtonEvent.Floor][a.ButtonEvent.Button] = true
			newMessage := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: responsibleElevator, OrderID: OrderNumber}
			go network.SendOrder(orderTX, nodeStatusRX, newMessage, a.ID, OrderNumber, requestRX, nodeStatusMap)
			OrderNumber++
			lastMessagesMap[a.ID] = a.RequestID

		case a := <-RXFloorReached:
			lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, lastMessagesMap)
			if lastMessageNumber == a.RequestID {
				continue
			}
			if a.ID != "" {
				index, _ := getOrAssignIndex(a.ID, config.IDToIndexMap)
				storedOrders = updateOrders(a.Orders, index, storedOrders)
				lastMessagesMap[a.ID] = a.RequestID
			}
		}
	}
}

func updateOrders(ordersFromNode [config.NFloors][config.NButtons]bool, elevator int, storedOrders [config.MElevators][config.NFloors][config.NButtons]bool) [config.MElevators][config.NFloors][config.NButtons]bool {
	newstoredOrders := storedOrders
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons; j++ {
			newstoredOrders[elevator][i][j] = ordersFromNode[i][j]
		}
	}
	return newstoredOrders
}

func getOrAssignIndex(ip string, idIndexMap map[string]int) (int, bool) {
	if index, exists := idIndexMap[ip]; exists {
		return index, true
	} else {
		idIndexMap[ip] = len(idIndexMap)
		return idIndexMap[ip], false
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

func searchMap(index int, idIndexMap map[string]int) string {
	for key, value := range idIndexMap {
		if value == index {
			return key
		}
	}
	return ""
}

func UpdateNodeMap(ID string, status network.SingleElevatorStatus, nodeMap map[string]network.SingleElevatorStatus) map[string]network.SingleElevatorStatus {
	if _, exists := nodeMap[ID]; exists {
		nodeMap[ID] = status
	} else {
		nodeMap[ID] = status
	}
	return nodeMap
}

func makeLightMatrix(ID string, storedOrders [config.MElevators][config.NFloors][config.NButtons]bool, idMap map[string]int) [config.NFloors][config.NButtons]bool {
	lightMatrix := [config.NFloors][config.NButtons]bool{}
	for floor := 0; floor < config.NFloors; floor++ {
		for button := 0; button < config.NButtons-1; button++ {
			for elev := 0; elev < config.MElevators; elev++ {
				if storedOrders[elev][floor][button] {
					lightMatrix[floor][button] = true
				}
			}
		}
	}
	for floor := 0; floor < config.NFloors; floor++ {
		lightMatrix[floor][2] = storedOrders[idMap[ID]][floor][2]
	}
	return lightMatrix
}

func distributeOrdersFromLostNode(lostNodeID string, storedOrders [config.MElevators][config.NFloors][config.NButtons]bool, idMap map[string]int, nodeMap map[string]network.SingleElevatorStatus, Peerlist peers.PeerUpdate) ([config.MElevators][config.NFloors][config.NButtons]bool, []network.Order) {
	distributedOrders := storedOrders
	lostNodeIndex, _ := getOrAssignIndex(lostNodeID, idMap)
	reassignedOrders := make([]network.Order, 0)
	lostOrders := storedOrders[lostNodeIndex]
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons-1; j++ {
			if lostOrders[i][j] {
				lostOrder := network.Order{ButtonEvent: elevator.ButtonEvent{Floor: i, Button: elevator.ButtonType(j)}, ResponisbleElevator: "", OrderID: OrderNumber}
				responsibleElevator := AssignOrder(lostOrder, Peerlist, nodeMap)
				lostOrder.ResponisbleElevator = responsibleElevator
				distributedOrders[lostNodeIndex][i][j] = false
				responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator, idMap)
				distributedOrders[responsibleElevatorIndex][i][j] = true
				reassignedOrders = append(reassignedOrders, lostOrder)
			}
		}
	}
	return distributedOrders, reassignedOrders
}
