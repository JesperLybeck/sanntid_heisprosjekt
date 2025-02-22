package main

import (
	"Network-go/network/bcast"
	//"Network-go/network/localip"
	//"Network-go/network/peers"
	"Sanntid/elevio"
	"fmt"
	
)



func main() {
	ch1 := make(chan elevio.ButtonEvent)
	//recieverHB := make(chan peers.PeerUpdate)
	//transmitterHB := make(chan bool)

	go bcast.Receiver(12055, ch1)
	for {
		select {
		case a := <-ch1:
			fmt.Println(a)
		}
	}
}