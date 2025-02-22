package main

import (
	"Network-go/network/bcast"
	//"Network-go/network/localip"
	//"Network-go/network/peers"
	"Sanntid/elevio"
	"fmt"
)

type order struct {
	Button elevio.ButtonEvent
	Id string
}


func main() {
	TXOrderCh := make(chan order)
	RXOrderCh := make(chan order)

	go bcast.Receiver(12070, RXOrderCh)
	go bcast.Transmitter(12070, TXOrderCh)
	for {
		select {
		case a := <-RXOrderCh:
			fmt.Println("Order received from main", a)
			if a.Id != "0" {
				fmt.Println("Order sent from main")
				fmt.Println(a)
				a.Id = "1"
				fmt.Println(a)
				TXOrderCh <- a
			}
		}
	}
}