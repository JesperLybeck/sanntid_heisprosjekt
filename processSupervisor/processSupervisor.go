package main

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Sanntid/fsm"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	print("supervisor running")
	nodeID := os.Getenv("ID")
	if nodeID == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			localIP = "DISCONNECTED"
		}
		nodeID = localIP
	}

	elevioPortNumber := os.Getenv("PORT") // Read the environment variable
	if elevioPortNumber == "" {
		elevioPortNumber = "15657" // Default value if the environment variable is not set
	}

	StartingAsPrimaryEnv := os.Getenv("STARTASPRIM")

	aliveSignalRX := make(chan fsm.SingleElevatorStatus)
	go bcast.Receiver(13059, aliveSignalRX)
	timer := time.NewTimer(8 * time.Second)

	print("NodeID: ", nodeID)
	print("Port: ", elevioPortNumber)
	print("Starting as primary: ", StartingAsPrimaryEnv)

	for {
		select {
		case status := <-aliveSignalRX:
			if status.ID == nodeID {
				timer.Reset(8 * time.Second)

			}

		case <-timer.C:
			runMainInParentDirectory(elevioPortNumber, nodeID, StartingAsPrimaryEnv)

			timer.Reset(8 * time.Second)
			print("Restarting main")
		}
	}
}

func runMainInParentDirectory(port string, nodeID string, startasprim string) {
	envVars := fmt.Sprintf("PORT=%s ID=%s STARTASPRIM=%s", port, nodeID, startasprim)

	// Create command that will exit when the process ends
	cmd := exec.Command("gnome-terminal", "--", "bash", "-c",
		fmt.Sprintf("%s exec go run main.go", envVars))
	err := cmd.Start()
	if err != nil {
		print("Error starting main.go:", err)
	}

}
