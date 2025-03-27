package pba

import (
	"Sanntid/config"
	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"strconv"
)

func RoleElection(ID string, primaryElection chan<- network.Election) {
	var primaryStatusRX = make(chan network.Status)
	go bcast.Receiver(13055, primaryStatusRX)
	LatestStatusFromPrimary := network.Status{}
	primID := ""

	mergedOrders := [config.MElevators][config.NFloors][config.NButtons]bool{}
	for {
		select {
		case p := <-primaryStatusRX:

			//fmt.Println("Backup received from primary")

			if p.TransmitterID != ID { //kun primary som sender meld. Hvis du får fra en annen, er det en primary
				intID, _ := strconv.Atoi(ID[len(ID)-2:])
				intTransmitterID, _ := strconv.Atoi(p.TransmitterID[len(ID)-2:])
				//Her mottar en primary melding fra en annen primary

				if intID > intTransmitterID {

					mergedOrders = mergeOrders(LatestStatusFromPrimary.Orders, p.Orders) //take over manager i stedet håndterer denne
					primID = ID

				} else if intID < intTransmitterID {

					primID = p.TransmitterID

				}

				electionResult := network.Election{PrimaryID: primID, MergedOrders: mergedOrders}

				primaryElection <- electionResult
			}

		}
	}
}
