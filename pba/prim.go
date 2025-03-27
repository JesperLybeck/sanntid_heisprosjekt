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

func Primary(id string, primaryElection <-chan network.Election, status network.Status) {
	storedOrders := status.Orders
	fmt.Print("stored orders in status,", storedOrders)

	primaryID := id
	backupID := ""
	nodeStatusMap := make(map[string]network.SingleElevatorStatus)
	previousprimaryID := status.PreviousPrimaryID
	takeOverInProgress := status.TakeOverInProgress
	latestPeerList := peers.PeerUpdate{}

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

	for {
		print("I am prim")

		select {
		case nodeUpdate := <-nodeStatusRX:

			nodeStatusMap = updateNodeMap(nodeUpdate.ID, nodeUpdate, nodeStatusMap)

			if takeOverInProgress {
				//do stuff
				print("takeover in progress")
				storedOrders = distributeOrdersFromLostNode(previousprimaryID, storedOrders, orderTX, nodeStatusRX, requestRX, config.IDToIndexMap, nodeStatusMap, latestPeerList)
				print("distributing orders from takeover", previousprimaryID)
				takeOverInProgress = false
			}
		case p := <-primaryElection:
			if id != p.PrimaryID {
				go Backup(id, primaryElection)
				return
			}

			primaryID = p.PrimaryID
			backupID = p.BackupID

		case p := <-peersRX:
			print("new peer update")

			fmt.Println("Peerupdate in prim, change of LatestPeerList")
			latestPeerList = p

			if backupID == "" && len(p.Peers) > 1 {
				for i := 0; i < len(p.Peers); i++ {
					if p.Peers[i] != id {
						backupID = p.Peers[i]
					}
				}
			}

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
							print("new order from restore cab calls")
							print("----------searchmapindex:-------->", searchMap(index, config.IDToIndexMap), "<------")
							go network.SendOrder(orderTX, nodeStatusRX, newOrder, searchMap(index, config.IDToIndexMap), OrderNumber, requestRX, nodeStatusMap)

							OrderNumber++

						}
					}

				}
			}
			fmt.Print("lost node", p.Lost)
			for i := 0; i < len(p.Lost); i++ {
				//alle som dør

				storedOrders = distributeOrdersFromLostNode(p.Lost[i], storedOrders, orderTX, nodeStatusRX, requestRX, config.IDToIndexMap, nodeStatusMap, latestPeerList)

				//hvis backup dør
				if p.Lost[i] == backupID {

					for j := 0; j < len(p.Peers); j++ {
						if p.Peers[j] != primaryID {
							backupID = p.Peers[j]
						} else {
							backupID = ""
						}
					}
				}
			}

		case <-ticker.C:
			//sending status to backup

			statusTX <- network.Status{
				TransmitterID:      id,
				Orders:             storedOrders,
				StatusID:           1,
				PreviousPrimaryID:  primaryID,
				AloneOnNetwork:     false,
				TakeOverInProgress: false,
				PeerList:           latestPeerList,
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
			fmt.Print("LastMessageNumber-->", lastMessageNumber, "--RequestID-->", a.RequestID, "---")

			order := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: a.ID, OrderID: OrderNumber}

			responsibleElevator := AssignOrder(order, latestPeerList, nodeStatusMap)
			print("responsible elevator", responsibleElevator)
			order.ResponisbleElevator = responsibleElevator

			responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator, config.IDToIndexMap)

			storedOrders[responsibleElevatorIndex][a.ButtonEvent.Floor][a.ButtonEvent.Button] = true
			//sent to backup in next status update

			newMessage := network.Order{ButtonEvent: a.ButtonEvent, ResponisbleElevator: responsibleElevator, OrderID: OrderNumber}

			//vi bør kanskje forsikre oss om at backup har lagret dette. Mulig vi må kreve ack fra backup, da vi bruker dette som knappelys garanti.

			go network.SendOrder(orderTX, nodeStatusRX, newMessage, id, OrderNumber, requestRX, nodeStatusMap)
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

func updateNodeMap(ID string, status network.SingleElevatorStatus, nodeMap map[string]network.SingleElevatorStatus) map[string]network.SingleElevatorStatus {
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

func distributeOrdersFromLostNode(lostNodeID string, storedOrders [config.MElevators][config.NFloors][config.NButtons]bool, orderTX chan<- network.Order, ackChan <-chan network.SingleElevatorStatus, resendChan chan network.Request, idMap map[string]int, nodeMap map[string]network.SingleElevatorStatus, Peerlist peers.PeerUpdate) [config.MElevators][config.NFloors][config.NButtons]bool {
	distributedOrders := storedOrders
	lostNodeIndex, _ := getOrAssignIndex(lostNodeID, idMap)
	print("dist orders from lost node", lostNodeID)
	lostOrders := storedOrders[lostNodeIndex]
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons-1; j++ {
			if lostOrders[i][j] {
				print("lost order found")
				lostOrder := network.Order{ButtonEvent: elevator.ButtonEvent{Floor: i, Button: elevator.ButtonType(j)}, ResponisbleElevator: "", OrderID: OrderNumber}
				fmt.Print("lost order", lostOrder, Peerlist, nodeMap)
				responsibleElevator := AssignOrder(lostOrder, Peerlist, nodeMap)
				print("responsible elevator", responsibleElevator)
				lostOrder.ResponisbleElevator = responsibleElevator

				distributedOrders[lostNodeIndex][i][j] = false
				print("clearing order from dead node orders")

				responsibleElevatorIndex, _ := getOrAssignIndex(responsibleElevator, idMap)
				print("responsible elevator", responsibleElevatorIndex)
				distributedOrders[responsibleElevatorIndex][i][j] = true
				print("new order from lost node")
				go network.SendOrder(orderTX, ackChan, lostOrder, responsibleElevator, OrderNumber, resendChan, nodeMap)
				OrderNumber++

			}
		}
	}
	return distributedOrders

}
