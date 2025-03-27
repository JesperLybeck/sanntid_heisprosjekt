package pba

import (
	"Sanntid/config"
	"Sanntid/network"
	"Sanntid/networkDriver/bcast"
	"strconv"
)

func RoleElection(ID string, backupSignal chan<- bool, startRoleElection <-chan bool, primaryElection chan<- network.Election) {
	for {
		print("Waiting for role election")
		<-startRoleElection
		var primaryStatusRX = make(chan network.Status)
		go bcast.Receiver(13055, primaryStatusRX)
		LatestStatusFromPrimary := network.Status{}
		primID := ""
		var storedOrders [config.MElevators][config.NFloors][config.NButtons]bool
		print("Started RoleElection")

		electionLoop:
		for {

			select {
			case p := <-primaryStatusRX:

				if p.TransmitterID != ID {
					intID, _ := strconv.Atoi(ID[len(ID)-2:])
					intTransmitterID, _ := strconv.Atoi(p.TransmitterID[len(ID)-2:])
					//Her mottar en primary melding fra en annen primary
					// Dette er bad med mye pakketap
					print("MyID", intID, "Transmitter", intTransmitterID)

					if intID > intTransmitterID {
						println("Min ID større")
						storedOrders = mergeOrders(LatestStatusFromPrimary.Orders, p.Orders) //take over manager i stedet håndterer denne
						primID = ID
					} else if intID < intTransmitterID {
						println("Min ID mindre")
						primID = p.TransmitterID
					}

					electionResult := network.Election{PrimaryID: primID, MergedOrders: storedOrders}
					primaryElection <- electionResult
					if primID != ID {
						break electionLoop
					}
				}
			}
		}
	}
}
