package network

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/networkDriver/bcast"
	"time"
)

func SendRequestUpdate(transmitterChan chan<- Request, message Request, idToIndexMap map[string]int) {
	primStatusRX := make(chan Status)
	go bcast.Receiver(13055, primStatusRX)
	sendingTicker := time.NewTicker(30 * time.Millisecond)
	messageTimer := time.NewTimer(5 * time.Second)
	defer sendingTicker.Stop()
	messagesSent := 0
	for {
		select {
		case <-sendingTicker.C:
			transmitterChan <- message
			messagesSent++

		case status := <-primStatusRX:
			floor := message.ButtonEvent.Floor
			button := message.ButtonEvent.Button
			j := idToIndexMap[message.ID]
			if button == elevator.BT_Cab {
				if (status.Orders[j][floor][button] == message.Orders[floor][button]) && messagesSent > 0 {

					return
				}
			} else {
				for i := 0; i < config.MElevators; i++ {
					if (status.Orders[i][floor][button] == message.Orders[floor][button]) && messagesSent > 0 {
						return
					}
				}
			}

		case <-messageTimer.C:
			return

		}
	}
}

func SendOrder(transmitterChan chan<- Order, ackChan <-chan SingleElevatorStatus, message Order, ID string, OrderID int, ResendChan chan<- Request, nodeStatusMap map[string]SingleElevatorStatus) {
	messageTimer := time.NewTimer(5 * time.Second)
	sendingTicker := time.NewTicker(30 * time.Millisecond)
	defer sendingTicker.Stop()
	messagesSent := 0
	for {
		select {
		case <-sendingTicker.C:
			messagesSent++
			transmitterChan <- message

		case status := <-ackChan:
			button := message.ButtonEvent.Button
			floor := message.ButtonEvent.Floor
			if message.ResponisbleElevator == status.ID && (status.Orders[floor][button] || (message.ButtonEvent.Floor == status.PrevFloor && messagesSent > 0)) {
				return
			}

		case <-messageTimer.C:
			RequestID := message.OrderID
			Reassign := Request{ID: ID, ButtonEvent: message.ButtonEvent, Orders: nodeStatusMap[ID].Orders, RequestID: RequestID}
			ResendChan <- Reassign
			return
		}
	}
}
