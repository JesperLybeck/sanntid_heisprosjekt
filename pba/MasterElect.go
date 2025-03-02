package pba

import (
	"Network-go/network/peers"
)

func AssignPrimary(ID string) {
	println("PeerUpdates started")
	peersRX := make(chan peers.PeerUpdate)

	go peers.Receiver(12055, peersRX)

	/*for {
		select {
		case p := <-peersRX:
			if len(p.Peers) == 1 {
				println("PrimaryID set to", p.Peers[0])
				fsm.PrimaryID = ID
				return
			}
		}
	}*/

}
