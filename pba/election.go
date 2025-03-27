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
	backupID := ""
	mergedOrders := [config.MElevators][config.NFloors][config.NButtons]bool{}
	for {
		select {
		case p := <-primaryStatusRX:

			//fmt.Println("Backup received from primary")
			if primID == "" {
				primID = p.TransmitterID
			} else {

				if p.TransmitterID != primID { //kun primary som sender meld. Hvis du får fra en annen, er det en primary
					intID, _ := strconv.Atoi(ID[len(ID)-2:])
					intTransmitterID, _ := strconv.Atoi(p.TransmitterID[len(ID)-2:])
					//Her mottar en primary melding fra en annen primary

					if intID > intTransmitterID {
						println("Min ID større")
						mergedOrders = mergeOrders(LatestStatusFromPrimary.Orders, p.Orders) //take over manager i stedet håndterer denne
						primID = ID
						backupID = p.TransmitterID

					} else if intID < intTransmitterID {
						println("Min ID mindre")
						primID = p.TransmitterID
						backupID = ID
					}

					electionResult := network.Election{TakeOverInProgress: false, LostNodeID: "", PrimaryID: primID, BackupID: backupID, MergedOrders: mergedOrders}
					primaryElection <- electionResult
				}
			} /*else {

				if p.TransmitterID != ID {
					PrimaryID = p.TransmitterID
					BackupID = ID
				}
				if p.ReceiverID == ID && BackupID != ID {
					BackupID = ID
				}

			}*/

		}
	}
}
