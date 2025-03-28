package pba

import (
	"Sanntid/elevator"
	"Sanntid/network"
	"Sanntid/networkDriver/peers"
	"math"
)

type CostTuple struct {
	Cost int
	ID   string
}

func AssignOrder(request network.Order, peerList peers.PeerUpdate, nodeStatus map[string]network.SingleElevatorStatus) string {
	for {
		costs := make([]CostTuple, len(peerList.Peers))
		if request.ButtonEvent.Button == elevator.BT_Cab {
			return request.ResponisbleElevator
		}
		for p := 0; p < len(peerList.Peers); p++ {
			peerStatus := nodeStatus[peerList.Peers[p]]
			costs[p].ID = peerList.Peers[p]
			distanceCost := (peerStatus.PrevFloor - request.ButtonEvent.Floor) * (peerStatus.PrevFloor - request.ButtonEvent.Floor)
			costs[p].Cost = distanceCost
		}
		responsibleElevator := argmin(costs)
		if responsibleElevator != "" {
			return responsibleElevator
		}
	}
}

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
