# Distributed elevator system

## Overview
This project is an distributed elevator system with heavy emphasis on fault tolerance. The project is Implemented in Go. It follows a modular design to manage M elevators and their operations, networking, process supervision.

## Project Structure
Below is an overview of the key modules in this project:

- **config**: Holds global constants which are used by all modules.
- **elevator**: Core logic for elevator movement, request handling, and state management.
- **network**: Manages communication between each module.
- **networkDriver**: Low-level networking functionalities using UDP.
- **pba**: Primary and backup assignment for each eleveator 
- **processSupervisor**: Monitors and manages system processes.

## How to run the program
Edit the run.sh file with the environment variables PORT, ID and STARTASPRIM. 
- **ID**:  When running the program for 3 elevators or less, set ID equal to "111", "222" or "333". 
- **PORT**:  Set PORT equal to "15657" when running on a physical elevator.
- **STARTASPRIM**:  Set STARTASPRIM = "true" for one elevator and STARTASPRIM = "false" for the rest

To run the program open a terminal window in the folder contatining the project. 
- Type ./run.sh in the terminal and a process supervisior window will open to monitor the given node as well as a window which runs main.go 
- If the node enters a failure state the supervisor will terminate main.go and start another. The new main stalls while it is in a failure state. 
