#!/bin/bash



# Set the variables
nodeID="111"
PORT=16000

STARTASPRIM=true

LASTNUMBER=${PORT: -1}
go build -o "elevator_${LASTNUMBER}" main.go

# Check if Go is installed
if ! command -v go &> /dev/null
then
    echo "Go could not be found. Please install Go and ensure it is in your PATH."
    exit
fi

# Function to start SimElevatorServer
start_sim_elevator_server() {
    echo "Starting SimElevatorServer..."
    gnome-terminal -- bash -c "simelevatorserver --port=$PORT; exec bash" &
    #gnome-terminal -- bash -c "elevatorserver; exec bash" &
}

# Function to start processSupervisor
start_process_supervisor() {
    echo "Starting processSupervisor..."
    gnome-terminal -- bash -c "env ID=$nodeID PORT=$PORT STARTASPRIM=$STARTASPRIM go run processSupervisor/processSupervisor.go; exec bash" &
    #gnome-terminal -- bash -c "env ID=$nodeID PORT=$PORT STARTASPRIM=$STARTASPRIM go run processSupervisor/processSupervisor.go; exec bash" &
}

# Start the SimElevatorServer
start_sim_elevator_server


# Start the processSupervisor
start_process_supervisor

echo "Both SimElevatorServer and processSupervisor have been started."