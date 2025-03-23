package pba

import (
	"Network-go/network/peers"
	"Sanntid/elevio"
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

func AssignOrder(request fsm.Order, latestPeerList peers.PeerUpdate) string {
	if len(latestPeerList.Peers) == 0 {
		panic("No avaialable elevators")
	}
	costs := make([]fsm.CostTuple, len(latestPeerList.Peers)) // costs for each elevator
	if request.ButtonEvent.Button == elevio.BT_Cab {
		return request.ResponisbleElevator
	}

	for p := 0; p < len(latestPeerList.Peers); p++ {
		peerStatus := fsm.NodeStatusMap[latestPeerList.Peers[p]]
		costs[p].ID = latestPeerList.Peers[p]
		distanceCost := (peerStatus.PrevFloor - request.ButtonEvent.Floor) * (peerStatus.PrevFloor - request.ButtonEvent.Floor)
		//Optional: add directional contribution to cost.

		costs[p].Cost = distanceCost // + directionCost

	}
	//fmt.Println("costs:", costs)
	responsibleElevator := argmin(costs)
	return responsibleElevator

}
