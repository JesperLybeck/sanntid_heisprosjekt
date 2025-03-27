package network

import (
	"Sanntid/config"
	"Sanntid/elevator"
	"Sanntid/networkDriver/bcast"

	"strconv"
	"time"
)

func SendRequestUpdate(transmitterChan chan<- Request, message Request, requestID int, idToIndexMap map[string]int) {

	primStatusRX := make(chan Status)
	go bcast.Receiver(13055, primStatusRX)

	sendingTicker := time.NewTicker(30 * time.Millisecond)
	messageTimer := time.NewTimer(10 * time.Second)

	defer sendingTicker.Stop()

	//dette betyr at andre noder kan acke ordre som ikke er til dem?

	messagesSent := 0

	print("---------Sending request update---------")

	for {
		select {
		case <-sendingTicker.C:

			transmitterChan <- message
			messagesSent++

		case status := <-primStatusRX: //kan dette skje på samme melding?

			floor := message.ButtonEvent.Floor
			button := message.ButtonEvent.Button
			//print("ID: ", message.ID, "index: ", IpToIndexMap[message.ID])
			j := idToIndexMap[message.ID]
			if button == elevator.BT_Cab {
				if (status.Orders[j][floor][button] == message.Orders[floor][button]) && messagesSent > 0 {
					print("--------- Request acked ---------")
					return
				}
			} else {
				for i := 0; i < config.MElevators; i++ {
					if (status.Orders[i][floor][button] == message.Orders[floor][button]) && messagesSent > 0 {

						print("--------- Request acked ---------")
						return
					}
				}
			}

		case <-messageTimer.C:
			print("No ack received for request, stopping transmission.")
			//vi trenger ikke å sende error her. vi kan anta bruker trykker på knappen på nytt.
			return

		}
	}
}

func SendOrder(transmitterChan chan<- Order, ackChan <-chan SingleElevatorStatus, message Order, ID string, OrderID int, ResendChan chan<- Request, nodeStatusMap map[string]SingleElevatorStatus) {
	messageTimer := time.NewTimer(5 * time.Second)
	sendingTicker := time.NewTicker(30 * time.Millisecond)

	defer sendingTicker.Stop()
	messagesSent := 0
	// er vi nødt til å acke ordre gitt i etasje vi allerede er i?
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
			Reassign := Request{ID: ID, ButtonEvent: message.ButtonEvent, RequestID: RequestID}
			ResendChan <- Reassign
			return

			//kan vi throwe en error her, som sørger for at ordren forsøkes håndtert på nytt? den kan da sendes til en annen node i stedet??

		}
	}
}

func incrementMessage(messageID string) string {
	nodeID := messageID[:3]
	messageNumber := messageID[3:]
	messageNumberInt, _ := strconv.Atoi(messageNumber)
	messageNumberInt++
	messageNumber = strconv.Itoa(messageNumberInt)
	return nodeID + messageNumber

}
