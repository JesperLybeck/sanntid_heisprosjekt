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
	storedOrders := initialState.StoredOrders

	nodeStatusMap := make(map[string]network.SingleElevatorStatus)
	previousprimaryID := initialState.PreviousPrimaryID
	takeOverInProgress := initialState.TakeOverInProgress
	latestPeerList := initialState.Peerlist

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

	ticker := time.NewTicker(100 * time.Millisecond)
	lightUpdateTicker := time.NewTicker(50 * time.Millisecond)

	var lastMessagesMap = make(map[string]int)

	if takeOverInProgress {
		//do stuff

		lostOrders := make([]network.Order, 0)
		storedOrders, lostOrders = distributeOrdersFromLostNode(previousprimaryID, storedOrders, config.IDToIndexMap, nodeStatusMap, latestPeerList)
		print("Lost orders: ", lostOrders)
		for order := 0; order < len(lostOrders); order++ {
			go network.SendOrder(orderTX, nodeStatusRX, lostOrders[order], id, OrderNumber, requestRX, nodeStatusMap)
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
				//go Backup(id, primaryElection,)
				//nedgradere til backup
				done <- true
				return
			}

		case p := <-peersRX:

			latestPeerList = p

			if p.New != "" {
				index, exists := getOrAssignIndex(p.New, config.IDToIndexMap)

				if exists {
					// Retrieve CAB calls.
					// kanskje vi kan lage en "fake" new order? Eventuelt om vi bør endre single elevator til å ikke være event basert, men heller "while requests in queue"
					//Hvis vi finner at det er lagret cab calls for denne heisen som ikke er gjort her i remote, så trigger vi en ny ordre .
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
				//alle som dør
				lostOrders := make([]network.Order, 0)
				storedOrders, lostOrders = distributeOrdersFromLostNode(p.Lost[i], storedOrders, config.IDToIndexMap, nodeStatusMap, latestPeerList)
				print("Lost orders: ", lostOrders)
				for order := 0; order < len(lostOrders); order++ {
					go network.SendOrder(orderTX, nodeStatusRX, lostOrders[order], id, OrderNumber, requestRX, nodeStatusMap)
					OrderNumber++
				}
			}

		case <-ticker.C:
			//sending status to backup

			statusTX <- network.Status{
				TransmitterID:      id,
				Orders:             storedOrders,
				StatusID:           1,
				AloneOnNetwork:     false,
				TakeOverInProgress: false,
			}
			//periodic light update to nodes.

			//when it is time to send light update:
			// for each node that is active:

		case <-lightUpdateTicker.C:

			for i := 0; i < len(config.IDToIndexMap); i++ {
				//compute the new lightmatrix given the stored orders.

				lightUpdate := makeLightMatrix(searchMap(i, config.IDToIndexMap), storedOrders, config.IDToIndexMap)
				//problem. denne oppdaterer kun hall light for 1 node av gangen, men denne oppdateringen må gå på alle.

				//if the new lightmatrix is different from the previous lights for the node:

				TXLightUpdates <- network.LightUpdate{LightArray: lightUpdate, ID: searchMap(i, config.IDToIndexMap)}
				//send out the updated lightmatrix to the node.
			}

		case a := <-requestRX:

			//Hall assignment
			/*if hallCallAssigned(a) {
				continue
			}*/
			//Update storedOrders

			lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, lastMessagesMap)
			if lastMessageNumber == a.RequestID {

				continue
			}
			order := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: a.ID, OrderID: OrderNumber}

			responsibleElevator := AssignOrder(order, latestPeerList, nodeStatusMap)

			order.ResponisbleElevator = responsibleElevator

			responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator, config.IDToIndexMap)

			storedOrders[responsibleElevatorIndex][a.ButtonEvent.Floor][a.ButtonEvent.Button] = true
			//sent to backup in next status update

			newMessage := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: responsibleElevator, OrderID: OrderNumber}

			//vi bør kanskje forsikre oss om at backup har lagret dette. Mulig vi må kreve ack fra backup, da vi bruker dette som knappelys garanti.

			go network.SendOrder(orderTX, nodeStatusRX, newMessage, id, OrderNumber, requestRX, nodeStatusMap)

			OrderNumber++
			lastMessagesMap[a.ID] = a.RequestID

		case a := <-RXFloorReached:

			lastMessageNumber, _ := getOrAssignMessageNumber(a.ID, lastMessagesMap)
			if lastMessageNumber == a.RequestID {

				continue
			}
			if a.ID != "" { //liker ikke dennne her):

				index, _ := getOrAssignIndex(a.ID, config.IDToIndexMap) //kan bli -1 hvis vi ikke er i mappet.

				storedOrders = updateOrders(a.Orders, index, storedOrders)

				lastMessagesMap[a.ID] = a.RequestID
			}

		}

	}

}

func updateOrders(ordersFromNode [config.NFloors][config.NButtons]bool, elevator int, storedOrders [config.MElevators][config.NFloors][config.NButtons]bool) [config.MElevators][config.NFloors][config.NButtons]bool {
	if elevator == -1 {

	}
	newstoredOrders := storedOrders

	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons; j++ {

			newstoredOrders[elevator][i][j] = ordersFromNode[i][j]
		}
	}

	// Copy existing orders
	/*for i := range storedOrders {
		for j := range storedOrders[i] {
			copy(newstoredOrders[i][j][:], storedOrders[i][j][:])
		}
	}
	newstoredOrders[elevator] = ordersFromNode*/

	return newstoredOrders
}
func getIndex(ip string, idIndexMap map[string]int) (int, bool) {

	if index, exists := idIndexMap[ip]; exists {

		return index, true
	} else {
		return -1, false
	}

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
		lightMatrix[floor][2] = storedOrders[idMap[ID]][floor][2] //setter cab lights inidividuelt
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
