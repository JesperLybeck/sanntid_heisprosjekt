package pba

import (
	"Sanntid/elevio"
	"Sanntid/fsm"
	"fmt"
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

func AssignOrder(request fsm.Order) string {
	print("assigning order")
	for {
		select {
			/*
			case update := <-peerCh:
				fsm.LatestPeerList = update
				fmt.Println("Peerupdate in assigner, change of LatestPeerList")
				*/
			default:
				fmt.Print("no peer update: ", fsm.LatestPeerList)
				costs := make([]fsm.CostTuple, len(fsm.LatestPeerList.Peers)) // costs for each elevator
				if request.ButtonEvent.Button == elevio.BT_Cab {
					return request.ResponisbleElevator
				}

				for p := 0; p < len(fsm.LatestPeerList.Peers); p++ {
					peerStatus := fsm.NodeStatusMap[fsm.LatestPeerList.Peers[p]]
					costs[p].ID = fsm.LatestPeerList.Peers[p]
					distanceCost := (peerStatus.PrevFloor - request.ButtonEvent.Floor) * (peerStatus.PrevFloor - request.ButtonEvent.Floor)
					//Optional: add directional contribution to cost.

					costs[p].Cost = distanceCost // + directionCost

				}
				//fmt.Println("costs:", costs)
				responsibleElevator := argmin(costs)
				print("Responsible elevator: ", responsibleElevator)
				if responsibleElevator != "" {

					return responsibleElevator
				}
		}
	}

}
