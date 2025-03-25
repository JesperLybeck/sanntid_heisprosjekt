package pba

import (
	"Network-go/network/bcast"
	"Sanntid/fsm"
	"strconv"
)

func RoleElection(ID string, primaryElection chan<- fsm.Election) {
	var primaryStatusRX = make(chan fsm.Status)
	go bcast.Receiver(13055, primaryStatusRX)
	LatestStatusFromPrimary := fsm.Status{}
	primID := ""
	backupID := ""
	for {
		select {
		case p := <-primaryStatusRX:

				//fmt.Println("Backup received from primary")

			if fsm.PrimaryID == ID && p.TransmitterID != ID {
				intID, _ := strconv.Atoi(ID[len(ID)-2:])
				intTransmitterID, _ := strconv.Atoi(p.TransmitterID[len(ID)-2:])
				//Her mottar en primary melding fra en annen primary
				// Dette er bad med mye pakketap
				print("MyID", intID, "Transmitter", intTransmitterID)

				fsm.LatestPeerList = p.Peerlist
					
				if intID > intTransmitterID {
					println("Min ID større")
					fsm.StoredOrders = mergeOrders(LatestStatusFromPrimary.Orders, p.Orders)
					primID = ID
					backupID = p.TransmitterID

				} else if intID < intTransmitterID {
					println("Min ID mindre")
					primID = p.TransmitterID
					backupID = ID
				}
					
				electionResult := fsm.Election{TakeOverInProgress: false, LostNodeID: "", PrimaryID: primID, BackupID: backupID}
				primaryElection <- electionResult
			} else {

				if p.TransmitterID != ID {
					fsm.PrimaryID = p.TransmitterID
					fsm.BackupID = ID
				}
				/*if p.ReceiverID == ID && fsm.BackupID != ID {
					fsm.BackupID = ID
				}*/

			}  

		}
	}
}
