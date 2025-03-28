package pba

import (
	"Sanntid/elevator"
	"Sanntid/network"
	"Sanntid/networkDriver/peers"
	"fmt"
	"math"
)

type CostTuple struct {
	Cost int
	ID   string
}

// min function to find the minimum value in an array
func argmin(arr []CostTuple) string {
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

func AssignOrder(request network.Order, peerList peers.PeerUpdate, nodeStatus map[string]network.SingleElevatorStatus) string {
	fmt.Print("||||", nodeStatus, "||||")
	fmt.Print("||||", peerList, "||||")
	for {
		costs := make([]CostTuple, len(peerList.Peers)) // costs for each elevator
		if request.ButtonEvent.Button == elevator.BT_Cab {
			return request.ResponisbleElevator
		}

		for p := 0; p < len(peerList.Peers); p++ {
			peerStatus := nodeStatus[peerList.Peers[p]]
			costs[p].ID = peerList.Peers[p]

			distanceCost := (peerStatus.PrevFloor - request.ButtonEvent.Floor) * (peerStatus.PrevFloor - request.ButtonEvent.Floor)
			//Optional: add directional contribution to cost.

			costs[p].Cost = distanceCost // + directionCost

		}
		//fmt.Println("costs:", costs)
		responsibleElevator := argmin(costs)

		if responsibleElevator != "" {

			return responsibleElevator
		}
	}

}
