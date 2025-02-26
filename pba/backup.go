package pba

import (
	"Network-go/network/bcast"
	"Sanntid/fsm"
	"fmt"
	"time"
)

var LatestStatusFromPrimary fsm.Status

func Backup(ID string) {
	for {
		if ID == fsm.BackupID {

			timeout := time.After(3 * time.Second) // Set timeout duration
			primaryStatusRX := make(chan fsm.Status)
			go bcast.Receiver(13055, primaryStatusRX)

			for {
				select {
				case p := <-primaryStatusRX:
					fmt.Println("Primary status received", p)
					timeout = time.After(5 * time.Second)

				case <-timeout:
					println("Primary status lost")

				}

			}
		}
	}
}
