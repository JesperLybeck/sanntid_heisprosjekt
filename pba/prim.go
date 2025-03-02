package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/fsm"
	"fmt"
	"time"
)

func Primary(ID string) {
	for {
		if ID == fsm.PrimaryID {

			statusTX := make(chan fsm.Status)
			orderTX := make(chan fsm.Order)
			orderRX := make(chan fsm.Order)

			//peerTX := make(chan bool)
			peersRX := make(chan peers.PeerUpdate)

			go peers.Receiver(12055, peersRX)
			go bcast.Transmitter(13055, statusTX)
			go bcast.Transmitter(13056, orderTX)
			go bcast.Receiver(13057, orderRX)

			ticker := time.NewTicker(2 * time.Second)

			for {
				select {
				case p := <-peersRX:
					if fsm.BackupID == "" && len(p.Peers) > 1 {
						for i := 0; i < len(p.Peers); i++ {
							if p.Peers[i] != ID {
								fsm.BackupID = p.Peers[i]
							}
						}
						
					}
					fmt.Println("Peer update", p.Peers)
					fmt.Println("New", p.New)
					fmt.Println("Lost", p.Lost)
					for i := 0; i < len(p.Lost); i++ {
						if p.Lost[i]==fsm.BackupID{
							println("Backup lost")
							for j := 0; j < len(p.Peers); j++ {
								if p.Peers[j]!= fsm.PrimaryID{
									fsm.BackupID = p.Peers[j]
								} else {
									fsm.BackupID = ""
								}
							}
						}
					}

				case <-ticker.C:

					statusTX <- fsm.Status{TransmitterID: ID, RecieverID: fsm.BackupID, Orders: [4][3]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}}
					
				/*
				case a <-orderRX:
					Hall assignment 
					change ID in order 
					orderTX <- a
				
					*/
				}
			}
		}
	}


}
