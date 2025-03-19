#!/bin/bash

# Ensure the script has execute permissions
chmod +x "$0"

# Set the variables
nodeID="1"
PORT=16001
STARTASPRIM=true

# Check if Go is installed
if ! command -v go &> /dev/null
then
    echo "Go could not be found. Please install Go and ensure it is in your PATH."
    exit
fi

# Function to start SimElevatorServer
start_sim_elevator_server() {
    echo "Starting SimElevatorServer..."
    gnome-terminal -- bash -c "simelevatorserver --port=$PORT; read" &
    #gnome-terminal -- bash -c "elevatorserver; read" &
}

# Function to start processSupervisor
start_process_supervisor() {
    echo "Starting processSupervisor..."
    gnome-terminal -- bash -c "env ID=$nodeID PORT=$PORT STARTASPRIM=$STARTASPRIM go run processSupervisor/processSupervisor.go; read" &
    #gnome-terminal -- bash -c "env ID=$nodeID STARTASPRIM=$STARTASPRIM go run processSupervisor/processSupervisor.go; read" &
}   

# Start the SimElevatorServer
start_sim_elevator_server

# Start the processSupervisor
start_process_supervisor

echo "Both SimElevatorServer and processSupervisor have been started."