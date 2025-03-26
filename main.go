package main

import (
	"Network-go/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"Sanntid/elevator"
	"Sanntid/fsm"
	"Sanntid/pba"

	"os"
	"time"
)

// ------------------------------Utility functions--------------------------------

func setHardwareEffects(E fsm.Elevator) {
	elevator.SetMotorDirection(E.Output.MotorDirection)
	elevator.SetDoorOpenLamp(E.Output.Door)
	elevator.SetFloorIndicator(E.Input.PrevFloor)

}

// ----------------------------ENVIRONMENT VARIABLES-----------------------------
var ID string
var startingAsPrimaryEnv string
var startingAsPrimary bool
var elevatorPortNumber string


func main() {
	var lastClearedButtons[]elevator.ButtonEvent
	//-----------------------------CHANNELS-----------------------------
	peerTX := make(chan bool)
	nodeStatusTX := make(chan fsm.SingleElevatorStatus) //strictly having both should be unnecessary.
	RequestToPrimTX := make(chan fsm.Request)
	OrderFromPrimRX := make(chan fsm.Order)
	OrderCompletedTX := make(chan fsm.Request)
	LightUpdateFromPrimRX := make(chan fsm.LightUpdate)
	peersRX := make(chan peers.PeerUpdate)

	buttonPressCh := make(chan elevator.ButtonEvent)
	floorReachedCh := make(chan int)

	primaryMerge := make(chan fsm.Election)

	//doorTimeoutCh := make(chan bool)

	//vi skiller mellom intern komunikasjon og kommunikasjon med primary.
	//channels for internal communication is denoted with "eventCh" channels for external comms are named "EventTX or RX"

	//-----------------------------TIMERS-----------------------------
	statusTicker := time.NewTicker(30 * time.Millisecond)

	//------------------------ASSIGNING ENVIRONMENT VARIABLES------------------------
	ID = os.Getenv("ID")
	if ID == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			localIP = "DISCONNECTED"
		}
		ID = localIP
	}

	startingAsPrimaryEnv = os.Getenv("STARTASPRIM")
	if startingAsPrimaryEnv == "true" {
		startingAsPrimary = true
	} else {
		startingAsPrimary = false
	}

	if startingAsPrimary {
		fsm.PrimaryID = ID
	}

	elevatorPortNumber = os.Getenv("PORT") // Read the environment variable
	if elevatorPortNumber == "" {
		elevatorPortNumber = "15657" // Default value if the environment variable is not set
	}
	//-----------------------------STATE MACHINE VARIABLES-----------------------------
	var elevator fsm.Elevator
	var prevLightMatrix [fsm.NFloors][fsm.NButtons]bool
	var NumRequests int = 1
	var lastOrderID int = 0

	//-----------------------------INITIALIZING ELEVATOR-----------------------------
	elevator.Init("localhost:"+elevatorPortNumber, fsm.NFloors) //when starting, the elevator goes up until it reaches a floor.

	for {
		elevator.SetMotorDirection(elevator.MD_Up)
		if elevator.GetFloor() != -1 {
			elevator.SetMotorDirection(elevator.MD_Stop)
			break
		}
		time.Sleep(fsm.OrderTimeout * time.Millisecond) // Add small delay between polls
	}
	for j := 0; j < 4; j++ {
		elevator.SetButtonLamp(elevator.BT_HallUp, j, false) //skrur av alle lys ved initsialisering. Nødvendig???
		elevator.SetButtonLamp(elevator.BT_HallDown, j, false)
		elevator.SetButtonLamp(elevator.BT_Cab, j, false)
	}
	//elevator state machine variables are initialized.
	elevator.State = fsm.Idle                                        //after initializing the elevator, it goes to the idle state.
	elevator.Input.LocalRequests = [fsm.NFloors][fsm.NButtons]bool{} // not strictly necessary, but...
	elevator.Output.LocalOrders = [fsm.NFloors][fsm.NButtons]bool{}
	elevator.Input.PrevFloor = elevator.GetFloor()
	elevator.DoorTimer = time.NewTimer(3 * time.Second)
	elevator.Output.Door = true
	elevator.OrderCompleteTimer = time.NewTimer(fsm.OrderTimeout * time.Second)
	elevator.ObstructionTimer = time.NewTimer(7 * time.Second)

	elevator.ObstructionTimer.Stop()

	elevator.OrderCompleteTimer.Stop()
	//-----------------------------GO ROUTINES-----------------------------
	go pba.RoleElection(ID, primaryMerge)
	go pba.Primary(ID, primaryMerge) //starting go routines for primary and backup.
	go pba.Backup(ID, primaryMerge)

	go elevator.PollButtons(buttonPressCh) //starting go routines for polling HW
	go elevator.PollFloorSensor(floorReachedCh)

	go bcast.Transmitter(13057, RequestToPrimTX) //starting go routines for network communication with other primary.
	go bcast.Receiver(13056, OrderFromPrimRX)
	go bcast.Transmitter(13058, OrderCompletedTX)
	go bcast.Transmitter(13059, nodeStatusTX)
	go bcast.Receiver(13060, LightUpdateFromPrimRX)
	go peers.Transmitter(12055, ID, peerTX)
	go peers.Receiver(12055, peersRX)

	//-----------------------------MAIN LOOP-----------------------------
	for {

		select {
		/*case primStatus := <-primStatusRX:
		//Hva om vi gjør en del av dette her, heller enn i pba?
		fsm.IpToIndexMap = primStatus.Map
		fsm.LatestPeerList = primStatus.Peerlist	*/

		case p := <-peersRX: //vi klarer oss vell uten denne, med casen over?
			// To register if alone on network and enter offline mode
			if len(p.Peers) == 0 {
				fsm.AloneOnNetwork = true
			} else {
				fsm.AloneOnNetwork = false
			}
		case lights := <-LightUpdateFromPrimRX:

			//when light update is received from primary, the node updates its own lights with the newest information.
			if (fsm.LightsDifferent(prevLightMatrix, lights.LightArray)) && lights.ID == ID {

				for i := range fsm.NButtons {
					for j := range fsm.NFloors {
						elevator.SetButtonLamp(elevator.ButtonType(i), j, lights.LightArray[j][i]) // vi kan vurdere om denne faktisk kan pakkes inn i en funksjon da vi gjør dette flere steder  koden.

					}

				}
				prevLightMatrix = lights.LightArray

			}

		case btnEvent := <-buttonPressCh: //case for å håndtere knappe trykk. Sender ordre til prim.			// Hvis heisen er i etasje n og får knappetrykk i n trenger man ikke å sende ordre til primary
			elevator.Input.LocalRequests[btnEvent.Floor][btnEvent.Button] = true
			requestToPrimary := fsm.Request{
				ButtonEvent: btnEvent,
				ID:          ID,
				TargetID:    fsm.PrimaryID,
				Orders:      elevator.Input.LocalRequests,
				RequestID:   NumRequests,
			}
			// ISSUE! when the order is delegated to a different node, we cant ack on

			go fsm.SendRequestUpdate(RequestToPrimTX, requestToPrimary, NumRequests)
			NumRequests++

			if fsm.AloneOnNetwork && btnEvent.Button == elevator.BT_Cab {
				offlineOrder := fsm.Order{ButtonEvent: btnEvent, ResponisbleElevator: ID}
				elevator = fsm.HandleNewOrder(offlineOrder, elevator) //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.
				elevator.SetButtonLamp(elevator.BT_Cab, btnEvent.Floor, true)
				setHardwareEffects(elevator)
				print("Offline order received")
			}

			//vi diskuterte om vi trengte å ha egen case for cab. Vi kom frem til at det ikke trengs fordi:
			//i: Hvis noden har kræsjet, tar vi ikke ordre.
			//ii: Hvis noden er uten nett, setter den seg selv til primary, den vil dermed ta ordre fra cab selv.

		case order := <-OrderFromPrimRX: // I denne casen mottar noden en ordre fra primary.

			if order.ResponisbleElevator != ID || order.ResponisbleElevator == ID && lastOrderID == order.OrderID { // hvis ordren er til en annen heis, ignorer.
				//denne kunne også strengt tatt gått inn i handle new Order functionen.
				//fmt.Print("anti spam order received from prim", order.ResponisbleElevator != ID, "------------------>", order.ResponisbleElevator, "<---->", ID)
				continue
			}

			//problem om heisen allerede er i etasjen ordren er i.
			//Da vil primary ikke få ack, fordi handle new order legger ikke til ordren i localOrders.
			//programmet terminerer ikke.
			elevator = fsm.HandleNewOrder(order, elevator)
			lastOrderID = order.OrderID //når vi mottar en ny ordre kaller vi på en pure function, som returnerer heisen i neste tidssteg.

			setHardwareEffects(elevator)

		case a := <-floorReachedCh:

			temp := elevator
			elevator = fsm.HandleFloorReached(a, elevator)
			lastClearedButtons = LastClearedButtons(temp,elevator)

			setHardwareEffects(elevator)

			if fsm.AloneOnNetwork {
				elevator.SetButtonLamp(elevator.BT_Cab, a, false)
			}

		case <-elevator.DoorTimer.C:

			elevator.DoorObstructed = elevator.GetObstruction()

			elevator = fsm.HandleDoorTimeout(elevator)

			setHardwareEffects(elevator)

			elevator.OrderCompleteTimer.Stop()
			print("sending order complete message")

			for i := range lastClearedButtons {
				orderMessage := fsm.Request{ButtonEvent: lastClearedButtons[i],
				ID:        ID,
				TargetID:  fsm.PrimaryID,
				Orders:    elevator.Output.LocalOrders,
				RequestID: NumRequests}

				go fsm.SendRequestUpdate(OrderCompletedTX, orderMessage, NumRequests)
				NumRequests++
			}

		case <-elevator.OrderCompleteTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, possible engine failure or faulty sensor")
		case <-elevator.ObstructionTimer.C:
			print("Node failed to complete order. throwing panic")
			panic("Node failed to complete order, door obstruction")
		case <-statusTicker.C:

			nodeStatusTX <- fsm.SingleElevatorStatus{ID: ID, PrevFloor: elevator.Input.PrevFloor, MotorDirection: elevator.Output.MotorDirection, Orders: elevator.Output.LocalOrders}

		}

	}

}

func LastClearedButtons(e fsm.Elevator, b fsm.Elevator) []elevator.ButtonEvent {
	lcb := []elevator.ButtonEvent{}
	order1 := e.Output.LocalOrders
	order2 := b.Output.LocalOrders
	for i := 0; i < fsm.NFloors; i++ {
		for j := 0; j < fsm.NButtons; j++ {
			if order1[i][j] != order2[i][j] {
				lcb = append(lcb, elevator.ButtonEvent{Floor: i, Button: elevator.ButtonType(j)})
			}
		}
	}
	return lcb
}