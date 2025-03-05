package pba

import (
	"Network-go/network/bcast"
	"Sanntid/fsm"
	"fmt"
	"time"
	"strconv"
)

var LatestStatusFromPrimary fsm.Status

func Backup(ID string) {
	var timeout = time.After(3 * time.Second) // Set timeout duration
	var primaryStatusRX = make(chan fsm.Status)
	go bcast.Receiver(13055, primaryStatusRX)
	LatestStatusFromPrimary := fsm.Status{}
	isBackup := false
	for {
		if !isBackup {
			select {
			case p := <-primaryStatusRX:
				if (p.TransmitterID == ID){
					LatestStatusFromPrimary = p
				}
				if fsm.PrimaryID == ID && p.TransmitterID != ID { 
					intID, _ := strconv.Atoi(ID)
					intTransmitterID, _ := strconv.Atoi(p.TransmitterID)
					//Her mottar en primary melding fra en annen primary
					if  intID > intTransmitterID {
						mergeOrders(LatestStatusFromPrimary.Orders, p.Orders)
						fsm.PrimaryID = ID
						fsm.BackupID = p.TransmitterID
					} else {
						fsm.PrimaryID = p.TransmitterID
						fsm.BackupID = ""

					}

				}

				
				if fsm.Version == p.Version {
					println("Status from primary", p.TransmitterID, "to", p.ReceiverID)
					fsm.PrimaryID = p.TransmitterID
					if p.ReceiverID == ID {
						fsm.BackupID = ID
						isBackup = true
					}
					timeout = time.After(3 * time.Second)
				}/* else if p.Version > fsm.Version {
					fmt.Println("Primary version higher. accepting new primary")
					fsm.Version = p.Version
					fsm.PrimaryID = p.TransmitterID
					timeout = time.After(3 * time.Second)

				}*/
				
			}
		}
		time.Sleep(500 * time.Millisecond)
		if fsm.BackupID == ID {

			select {
			case p := <-primaryStatusRX:

				println("BackupID: ", fsm.BackupID, "My ID:", ID, "PrimaryID: ", fsm.PrimaryID)
				LatestStatusFromPrimary = p
				timeout = time.After(3 * time.Second)

			case <-timeout:
				fmt.Println("Primary timed out")
				fsm.Version++
				fsm.PrimaryID = ID
				fsm.BackupID = ""
				isBackup = false
			}
		}
	}

}

func mergeOrders(orders1 [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool, orders2 [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool) [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool {
	var mergedOrders [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons; j++ {
			for k := 0; k < fsm.MElevators; i++ {
				if orders1[i][j][k] || orders2[i][j][k] {
					mergedOrders[i][j][k] = true
				}
			}
		}
	}
	return mergedOrders
}