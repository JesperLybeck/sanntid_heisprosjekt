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
                if p.RecieverID == ID {
                    fsm.BackupID = ID
                    isBackup = true
                }
            }
        }
		if fsm.BackupID == ID {
			select {
			case p := <-primaryStatusRX:
				LatestStatusFromPrimary = p
			case <-timeout:
				fmt.Println("Primary timed out")
				fsm.PrimaryID = ID

				continue
			}	
		}
	}
}