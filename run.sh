#!/bin/bash



# Set the variables
nodeID="33"
PORT=1603
STARTASPRIM=false
LASTNUMBER=${PORT: -1}
USE_SIM_ELEVATOR_SERVER=true
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
}

# Function to startElevatorServer
start_elevator_server() {
    echo "Starting ElevatorServer..."
    gnome-terminal -- bash -c "elevatorserver; exec bash" &
}

# Function to start processSupervisor
start_process_supervisor() {
    echo "Starting processSupervisor..."
    gnome-terminal -- bash -c "env ID=$nodeID PORT=$PORT STARTASPRIM=$STARTASPRIM go run processSupervisor/processSupervisor.go; exec bash" &
}

# Start desired server
if [ "$USE_SIM_ELEVATOR_SERVER" = true ]; then
    start_sim_elevator_server
else
    start_elevator_server
fi

# Start the processSupervisor
start_process_supervisor

echo "Both server and processSupervisor have been started."