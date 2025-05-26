# Job Worker Service: Application Design Document

## 1. Summary

The Job Worker is a distributed system for executing and managing computational jobs across host machines. 
It provides a gRPC-based service with resource management capabilities using Linux cgroups, real-time log streaming, and a comprehensive CLI client for job management operations.

## 2. System Overview

### 2.1 Purpose

The Job Worker service provides a controlled environment for running arbitrary Linux processes with:

- Fine-grained resource limitations (CPU, memory, I/O)
- Process isolation and management
- Output capture and streaming
- GRPC API for job control
- MTLS

### 2.2 Key Features

- **Process Execution**: Run any executable program available on the host machine (Not Stdin required program)
- **Resource Control**: Limit CPU, memory, and disk I/O using cgroups
- **Child Process Management**: Properly handle and terminate child processes
- **Output Capture**: Stream process output in real-time using gRPC Stream
- **Job Lifecycle Management**: Create, monitor, stop, and clean up jobs
- **GRPC API**: GRPC endpoints for job management
- **MTLS**: using mTLS for authentication between client and server
- **Graceful Shutdown**: Clean termination of running jobs on service shutdown, resources released on job termination
- **CLI**: Interact with worker application using gRPC client 

## 3. Architecture

## 4. Data Model

## 5. Authentication (mTLS)

## 6. API Specification

### 6.1 Endpoints

- Create
- Stream
- Get
- STOP
- List

#### 5.1.1 Create Job

#### 5.1.2 Stream Job's output

#### 5.1.3 Get Job's Info

#### 5.1.4 Stop the Running Job

#### 5.1.5 List of All Jobs


## 6. Resource Management

### 6.1 CPU Control

The service uses cgroups v2 to limit CPU usage by setting the `cpu.max` value. The limit is specified as a percentage of
one CPU core.

### 6.2 Memory Control

Memory usage is limited by setting the `memory.max` value in the job's cgroup:

### 6.3 I/O Control

I/O bandwidth is controlled using the `io.max` setting, which limits read and write operations:

## 7. Output Management

### 7.1 Output Capture

Process output is captured using two complementary methods:

**In-memory buffer**: Job output is kept in memory. This approach provides efficient access to output data.

### 7.2 Real-time Streaming

The service uses gRPC Stream for real-time outputs:

1. Client establishes Stream connection
2. Server continuously monitors job output
3. New output lines are sent to client as they appear
4. Connection terminates when job completes or client disconnects
5. Client Disconnection will not affect the running job in host

## 8. Concurrency Design

### 8.1 Thread Safety

The service is designed for concurrent operation with multiple active clients:

### 8.2 Connection Management

- Each streaming client connection is handled in a separate goroutine
- Client disconnections are properly detected and resources released
- Client disconnections are not affect the job in the host

## 9. Error Handling

- Network errors are handled with appropriate GRPC status codes
- Resource unavailability (e.g., cgroups) is detected early

## 10. Deployment Considerations

### 10.1 System Requirements

- Linux kernel with cgroups v2 support (5.0+)
- Proper permissions to create and manage cgroups
- Recommended: 2+ CPU cores, 4GB+ RAM

## 11. CLI

### 11.1 Creating a job:

### 11.2 Streaming job logs:

### 11.3 Getting job details:

### 11.4 Stop running job:

### 11.5 List of all jobs:

