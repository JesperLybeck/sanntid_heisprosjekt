package pba

import (
	"Network-go/network/bcast"
	"Sanntid/fsm"
	"fmt"
	"strconv"
	"time"
	"unsafe"
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

				if fsm.PrimaryID == ID && p.TransmitterID != ID {
					intID, _ := strconv.Atoi(ID[len(ID)-2:])
					intTransmitterID, _ := strconv.Atoi(p.TransmitterID[len(ID)-2:])
					//Her mottar en primary melding fra en annen primary
					print("MyID", intID, "Transmitter", intTransmitterID)
					if intID > intTransmitterID {
						println("Min ID større")
						fsm.StoredOrders = mergeOrders(LatestStatusFromPrimary.Orders, p.Orders)
						fsm.PrimaryID = ID
						fsm.BackupID = p.TransmitterID

					} else if intID < intTransmitterID {
						println("Min ID mindre")
						fsm.PrimaryID = p.TransmitterID
						fsm.BackupID = ""
					}

				} else {

					if p.TransmitterID != ID {
						fsm.PrimaryID = p.TransmitterID
					}
					if p.ReceiverID == ID {
						fsm.BackupID = ID
						isBackup = true
					}

					timeout = time.After(3 * time.Second)
				} /* else if p.Version > fsm.Version {
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
				size := unsafe.Sizeof(p)
				fmt.Printf("Size of Status struct: %d bytes\n", size)

				LatestStatusFromPrimary = p
				fsm.StoredOrders = p.Orders
				fsm.IpToIndexMap = p.Map
				fsm.Version = p.Version
				fmt.Print(fsm.IpToIndexMap)

				timeout = time.After(3 * time.Second)

			case <-timeout:
				fmt.Println("Primary timed out")

				fsm.Version++
				fsm.PreviousPrimaryID = fsm.PrimaryID
				fsm.PrimaryID = ID
				fsm.BackupID = ""
				isBackup = false
				fsm.TakeOverInProgress = true

			}
		}
	}

}

func mergeOrders(orders1 [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool, orders2 [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool) [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool {
	var mergedOrders [fsm.NFloors][fsm.NButtons][fsm.MElevators]bool
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons; j++ {
			for k := 0; k < fsm.MElevators; k++ {
				if orders1[i][j][k] || orders2[i][j][k] {
					mergedOrders[i][j][k] = true
				}
			}
		}
	}
	return mergedOrders
}
