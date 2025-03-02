package pba

import (
	"Network-go/network/bcast"
	"Sanntid/fsm"
	"fmt"
	"time"
)

var LatestStatusFromPrimary fsm.Status

func Backup(ID string) {
	var timeout = time.After(3 * time.Second) // Set timeout duration
	var primaryStatusRX = make(chan fsm.Status)
	go bcast.Receiver(13055, primaryStatusRX)
	isBackup := false
	for {
		if !isBackup {
			select {
			case p := <-primaryStatusRX:
				fmt.Println("fsm ver", fsm.Version, "p ver", p.Version)
				if fsm.Version == p.Version {
					println("Status from primary", p.TransmitterID, "to", p.ReceiverID)
					fsm.PrimaryID = p.TransmitterID
					if p.ReceiverID == ID {
						fsm.BackupID = ID
						isBackup = true
					}
					timeout = time.After(3 * time.Second)
				} else if p.Version > fsm.Version {
					fmt.Println("Primary version higher. accepting new primary")
					fsm.Version = p.Version
					fsm.PrimaryID = p.TransmitterID
					timeout = time.After(3 * time.Second)

				}
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
			}
		}
	}

}
