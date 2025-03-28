package pba

import (
	"Sanntid/config"
	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"strconv"
)

func PrimaryElection(ID string, primaryElection chan<- network.Election) {
	var primaryStatusRX = make(chan network.Status)

	var latestStatusFromPrimary = network.Status{}
	var mergedOrders = [config.MElevators][config.NFloors][config.NButtons]bool{}

	go bcast.Receiver(13055, primaryStatusRX)

	for {
		select {
		case p := <-primaryStatusRX:
			if p.TransmitterID != ID {
				mergedOrders = mergeOrders(p.Orders, latestStatusFromPrimary.Orders)
				electionResult := network.Election{PrimaryID: decidePrim(p.TransmitterID, ID), MergedOrders: mergedOrders}
				primaryElection <- electionResult
			} else {
				latestStatusFromPrimary.Orders = p.Orders
			}
		}
	}
}

func decidePrim(transmitterID string, ID string) string {
	primID := ""
	intID, _ := strconv.Atoi(ID[len(ID)-2:])
	intTransmitterID, _ := strconv.Atoi(transmitterID[len(ID)-2:])
	if intID > intTransmitterID {
		primID = ID
	} else if intID < intTransmitterID {
		primID = transmitterID
	}
	return primID
}

func mergeOrders(orders1 [config.MElevators][config.NFloors][config.NButtons]bool, orders2 [config.MElevators][config.NFloors][config.NButtons]bool) [config.MElevators][config.NFloors][config.NButtons]bool {
	var mergedOrders [config.MElevators][config.NFloors][config.NButtons]bool
	for i := 0; i < config.NFloors; i++ {
		for j := 0; j < config.NButtons-1; j++ {
			for k := 0; k < config.MElevators; k++ {
				if orders1[k][i][j] || orders2[k][i][j] {
					mergedOrders[k][i][j] = true
				}
			}
		}
	}
	return mergedOrders
}
