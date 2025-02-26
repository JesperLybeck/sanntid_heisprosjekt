package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/fsm"
	"fmt"
	"time"
)

func Primary(ID string) {
	if ID == fsm.PrimaryID {

		statusTX := make(chan fsm.Status)

		//peerTX := make(chan bool)
		peersRX := make(chan peers.PeerUpdate)

		go peers.Receiver(12055, peersRX)
		go bcast.Transmitter(13055, statusTX)
		ticker := time.NewTicker(2 * time.Second)

		for {
			select {
			case p := <-peersRX:
				fmt.Println("Peer update", p.Peers)
				fmt.Println("New", p.New)
				fmt.Println("Lost", p.Lost)
			case <-ticker.C:
				statusTX <- fsm.Status{ID: ID, Orders: [4][3]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}}
			}
		}
	}

}
