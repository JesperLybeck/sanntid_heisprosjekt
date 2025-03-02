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
				println("Status from primary", p.TransmitterID, "to", p.RecieverID)
				fsm.PrimaryID = p.TransmitterID
                if p.RecieverID == ID {
                    fsm.BackupID = ID
                    isBackup = true
                }
				timeout = time.After(3 * time.Second)
            }
        }
		time.Sleep(500 * time.Millisecond)
		if fsm.BackupID == ID {
			
			select {
			case p := <-primaryStatusRX:
				println("BackupID: ", fsm.BackupID,"My ID:", ID , "PrimaryID: ", fsm.PrimaryID)
				LatestStatusFromPrimary = p
				timeout = time.After(3 * time.Second)

			case <-timeout:
				fmt.Println("Primary timed out")
				fsm.PrimaryID = ID
				fsm.BackupID = ""
			}	
		}
	}
}