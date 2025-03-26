package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	nodeID := os.Getenv("ID")
	startingAsPrimaryEnv := os.Getenv("STARTASPRIM")
	fmt.Print("port ", port, "ID", nodeID, "STARTASPRIM", startingAsPrimaryEnv)

	aliveTimer := time.NewTimer(6 * time.Second)
	aliveChannel := make(chan bool)
	lastDigit := string(port[len(port)-1])
	go processAlive(aliveChannel, "elevator_"+lastDigit)
	for {
		select {
		case <-aliveChannel:
			aliveTimer.Reset(6 * time.Second)

		case <-aliveTimer.C:
			reviveProcess(port, nodeID, startingAsPrimaryEnv)
			aliveTimer.Reset(6 * time.Second)
		}
	}
}

func processAlive(aliveChannel chan bool, processName string) {

	for {
		cmd := exec.Command("pgrep", processName)
		err := cmd.Run()
		if err == nil {
			print("process alive")
			aliveChannel <- true

		} else {
			print("process ", processName, " is dead")
		}

		time.Sleep(1 * time.Second)

	}
}

func reviveProcess(port string, nodeID string, startingAsPrimaryEnv string) {
	lastDigit := string(port[len(port)-1])   // Extract the last digit of the port
	processName := "./elevator_" + lastDigit // Construct the process name dynamically
	cmd := exec.Command("gnome-terminal", "--", "bash", "-c", processName+"; exec bash")
	cmd.Env = append(os.Environ(), "PORT="+port, "ID="+nodeID, "STARTASPRIM="+startingAsPrimaryEnv)
	err := cmd.Start()
	if err != nil {
		print("Error starting process:", err)
	} else {
		print("Revived process:", processName)
	}
}
