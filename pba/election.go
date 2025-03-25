package pba

import (
	"Network-go/network/bcast"
	"Sanntid/fsm"
	"fmt"
	"strconv"
)

func RoleElection(ID string, primaryElection chan<- fsm.Election) {
	var primaryStatusRX = make(chan fsm.Status)
	go bcast.Receiver(13055, primaryStatusRX)
	print("Role election")
	LatestStatusFromPrimary := fsm.Status{}
	primID := ""
	backupID := ""
	for {
		if fsm.BackupID != ID {
			select {
			case p := <-primaryStatusRX:
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
						//primaryElection <- fsm.Election{TakeOverInProgress: false, LostNodeID: "", PrimaryID: ID, BackupID: ""}
						primID = ID
						backupID = p.TransmitterID
						

					} else if intID < intTransmitterID {
						println("Min ID mindre")
						/*
						fsm.PrimaryID = p.TransmitterID
						fsm.BackupID = ""
						*/
						//primaryElection <- fsm.Election{TakeOverInProgress: false, LostNodeID: "", PrimaryID: "", BackupID: ""}
						primID = p.TransmitterID
						backupID = ID
						
					}
					electionResult := fsm.Election{TakeOverInProgress: false, LostNodeID: "", PrimaryID: primID, BackupID: backupID}
					fmt.Print(electionResult)
					primaryElection <- fsm.Election{TakeOverInProgress: false, LostNodeID: "", PrimaryID: primID, BackupID: backupID}
					
					

				} else {

					if p.TransmitterID != ID {
						fsm.PrimaryID = p.TransmitterID
					}
					if p.ReceiverID == ID {
						fsm.BackupID = ID
					}
				} 
			}
		}
	}
}