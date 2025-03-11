package pba

import (
	"Network-go/network/peers"
	"Sanntid/fsm"
	"math"
)

// min function to find the minimum value in an array
func argmin(arr []fsm.CostTuple) string {
	minVal := math.MaxInt64
	minID := ""
	for _, value := range arr {
		if value.Cost < minVal {
			minVal = value.Cost
			minID = value.ID
		}
	}
	return minID
}

// indexOf function to find the index of the minimum value
func indexOf(arr []int, value int) int {
	for i, v := range arr {
		if v == value {
			return i
		}
	}
	return -1 // Return -1 if the value is not found
}

func AssignRequest(request fsm.Order, latestPeerList peers.PeerUpdate) string {
	costs := make([]fsm.CostTuple, len(latestPeerList.Peers)) // costs for each elevator

	for p := 0; p < len(latestPeerList.Peers); p++ {
		peerStatus := fsm.NodeStatusMap[latestPeerList.Peers[p]]
		costs[p].ID = latestPeerList.Peers[p]
		distanceCost := (peerStatus.PrevFloor - request.ButtonEvent.Floor) * (peerStatus.PrevFloor - request.ButtonEvent.Floor)
		/*directionCost := 0
		if peerStatus.MotorDirection == elevio.MD_Up && request.ButtonEvent.Button == elevio.BT_HallUp {
			directionCost = 0
		} else if peerStatus.MotorDirection == elevio.MD_Down && request.ButtonEvent.Button == elevio.BT_HallDown {
			directionCost = 0
		} else if peerStatus.MotorDirection == elevio.MD_Stop {
			directionCost = 1
		} else {
			directionCost = 5
		}*/

		costs[p].Cost = distanceCost // + directionCost

	}
	//fmt.Println("costs:", costs)
	responsibleElevator := argmin(costs)
	return responsibleElevator

}
