package pba

import (
	"Network-go/network/bcast"
	"Network-go/network/peers"
	"Sanntid/fsm"
)

func DecideRole(ID string) {
	println("PeerUpdates started")
	peersRX := make(chan peers.PeerUpdate)

	go peers.Receiver(12055, peersRX)

	for {
		select {
		case p := <-peersRX:
			if len(p.Peers) == 1 {
				println("PrimaryID set to", p.Peers[0])
				fsm.PrimaryID = ID
				return
			}
			if len(p.Peers) == 2 {
				fsm.BackupID = ID
				return
			}

		}
	}
}

func CheckRoles(ID string) {
	statusRX := make(chan fsm.Status)
	go bcast.Receiver(13055, statusRX)
	for {
		select {
		case s := <-statusRX:
			if s.Role == "Primary" {
				println("PrimaryID set to", s.ID)
				fsm.PrimaryID = s.ID
				return
			}
			if s.Role == "Backup" {
				fsm.BackupID = s.ID
				return
			}

		}
	}
}
