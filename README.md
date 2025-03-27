# Controlling M ELevators across N Floors

## Overview
This project is an elevator system implemented in Go. It follows a modular design to manage elevator operations, networking, process supervision, and more.

## Project Structure
Below is an overview of the key modules in this project:

- **config**: Handles configuration settings for the system.
- **elevator**: Core logic for elevator movement, request handling, and state management.
- **network**: Manages communication between different system components.
- **networkDriver**: Low-level networking functionalities for message passing.
- **pba**: (Provide a brief description of this module's function)
- **processSupervisor**: Monitors and manages system processes to ensure fault tolerance.