package pba

import (
	"Network-go/network/peers"
	"Sanntid/fsm"
)

func PeerUpdates() {
	println("PeerUpdates started")
	peersRX := make(chan peers.PeerUpdate)
	go peers.Receiver(12055, peersRX)

	for {
		select {
		case p := <-peersRX:
			if len(p.Peers) == 1 {
				println("PrimaryID set to", p.Peers[0])
				fsm.PrimaryID = p.Peers[0]
				return

			} else if len(p.Peers) == 2 {
				println("BackupID set to", p.Peers[1])
				fsm.BackupID = p.Peers[1]
				return
			} else {
				return
			}
		}
	}
}
