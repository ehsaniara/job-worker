# Worker

[![Tests](https://github.com/ehsaniara/job-worker/actions/workflows/ci.yml/badge.svg)](https://github.com/ehsaniara/job-worker/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ehsaniara/job-worker)](https://goreportcard.com/report/github.com/ehsaniara/job-worker)
[![Go Version](https://img.shields.io/github/go-mod/go-version/ehsaniara/job-worker)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/release/ehsaniara/job-worker.svg)](https://github.com/ehsaniara/job-worker/releases/latest)

![worker-thum.png](docs/worker-thum.png)

**Worker** is a high-performance distributed job execution system built in Go that provides secure remote command
execution with comprehensive resource management and real-time monitoring capabilities. The system uses a sophisticated
single-binary architecture with gRPC-based communication, mutual TLS authentication, and role-based authorization,
allowing administrators to securely execute arbitrary commands on remote Linux systems while enforcing strict CPU,
memory, and I/O bandwidth limits through Linux cgroups v2.

> **Key Innovation**: Single binary with dual execution modes - the same binary acts as both the gRPC server and the job
> initialization process, providing complete job isolation with PID, mount, IPC, and UTS namespaces while maintaining
> host
> networking for maximum compatibility and performance.

## ✨ Key Features

### 🎛️ **Advanced Resource Management**

- **Linux Cgroups v2**: CPU, memory, and I/O bandwidth limiting
- **Namespace Isolation**: PID, mount, IPC, and UTS isolation per job
- **Host Networking**: Jobs use host network interface for maximum compatibility
- **Resource Enforcement**: Hard limits with automatic cleanup

### 📡 **Real-Time Operations**

- **Live Log Streaming**: Real-time output streaming with buffering
- **Job Lifecycle Management**: Create, monitor, stop, and cleanup jobs
- **High-Performance gRPC**: Protocol buffer serialization with configurable limits
- **Comprehensive Monitoring**: Job status tracking and resource usage monitoring

## 📖 Usage Examples

### Basic Job Management

```bash
# Create simple jobs (all use host networking)
./cli run echo "Hello, World!"
./cli run python3 script.py
./cli run bash -c "for i in {1..10}; do echo 'Processing $i'; sleep 1; done"

# List all jobs with status
./cli list

# Get detailed job information
./cli get <job-id>

# Stop a running job
./cli stop <job-id>

# Stream live job output
./cli log <job-id>
```

### Resource-Limited Jobs

```bash
# Limit CPU usage to 50% of one core
./cli run --max-cpu=50 python3 heavy_computation.py

# Limit memory usage to 512 MB
./cli run --max-memory=512 java -jar memory-intensive-app.jar

# Limit I/O bandwidth (bytes per second)
./cli run --max-iobps=1000000 dd if=/dev/zero of=/tmp/test bs=1M count=100

# Combine multiple resource limits
./cli run --max-cpu=25 --max-memory=256 --max-iobps=1000000 intensive-task.sh
```

### Network-Enabled Jobs

All jobs share the host network interface and can communicate with external services:

```bash
# Start a web server (accessible on host port 8080)
./cli run --max-cpu=100 --max-memory=1024 python3 -c "
import socket
import time
s = socket.socket()
s.bind(('0.0.0.0', 8080))
s.listen(5)
print('Server listening on host port 8080')
while True:
    print('Server running...')
    time.sleep(10)
"

# Access external APIs
./cli run curl -s https://api.github.com/users/octocat

# Database connections
./cli run mysql -h remote-db.com -u user -p database
```

## 🏗️ Architecture

### Process Flow

1. **Server Mode**: Handles gRPC requests, manages job lifecycle
2. **Job Creation**: Server launches itself in init mode for each job
3. **Init Mode**: Sets up isolation, executes user command
4. **Isolation**: PID, mount, IPC, UTS namespaces (host networking)
5. **Monitoring**: Server monitors job processes and streams output
6. **Cleanup**: Automatic resource cleanup when jobs complete

### Core Components

**Execution Modes (`internal/modes/`)**

- **Server Mode**: gRPC server, job management, authentication
- **Init Mode**: Job isolation setup, command execution, namespace management

**Platform Abstraction (`pkg/platform/`)**

- Cross-platform compatibility layer
- Linux-specific implementations for namespaces and cgroups
- Fallback implementations for development

**Resource Management (`internal/worker/core/linux/resource/`)**

- Linux cgroups v2 integration
- CPU, memory, and I/O bandwidth limiting
- Automatic resource cleanup

**Process Management (`internal/worker/core/linux/process/`)**

- Process lifecycle management
- Namespace setup and isolation
- Signal handling and cleanup

## 🔐 Security & Authentication

### Certificate-Based Authentication

The system implements mutual TLS with role-based access control:

| Role       | Run Jobs | Status Jobs | Stop Jobs | Logs/Stream | Certificate OU |
|------------|----------|-------------|-----------|-------------|----------------|
| **Admin**  | ✅        | ✅           | ✅         | ✅           | `OU=admin`     |
| **Viewer** | ❌        | ✅           | ❌         | ✅           | `OU=viewer`    |

### Passwordless Certificate Management

```bash
# Generate all certificates (no password prompts)
make certs

# Setup admin CLI certificates
make setup-cli

# Setup viewer certificates
bash scripts/setup_cli_certs.sh --type viewer

# Copy certificates from remote server
bash scripts/setup_cli_certs.sh --remote --server your-server.com
```

### Process Isolation

Jobs run in isolated namespaces with the following characteristics:

- **PID Namespace**: Jobs see only their own process tree
- **Mount Namespace**: Isolated filesystem view with `/proc` remounting
- **IPC Namespace**: Separate inter-process communication
- **UTS Namespace**: Isolated hostname and domain name
- **Network**: Uses host networking (no network namespace isolation)
- **Cgroup Namespace**: Isolated resource management view

## ⚙️ Configuration

### Environment Variables

```bash
# Server Configuration
export WORKER_MODE="server"              # Execution mode (server|init)
export LOG_LEVEL="INFO"                  # DEBUG, INFO, WARN, ERROR

# Default Resource Limits
export WORKER_DEFAULT_CPU=100            # 100% of one CPU core
export WORKER_DEFAULT_MEMORY=512         # 512 MB memory limit
export WORKER_DEFAULT_IO=0               # Unlimited I/O (0 = disabled)

# gRPC Configuration
export MAX_RECV_MSG_SIZE=4194304         # 4MB max receive message size
export MAX_SEND_MSG_SIZE=4194304         # 4MB max send message size
```

### Systemd Service Configuration

The worker service runs with proper cgroup delegation:

```ini
[Service]
ExecStart=/opt/worker/worker
User=root
Delegate=yes
DelegateControllers=cpu memory io pids
Environment="WORKER_MODE=server"
```

## 📋 API Reference

### gRPC Service

```protobuf
service JobService {
  rpc RunJob(RunJobReq) returns (RunJobRes);
  rpc GetJobStatus(GetJobStatusReq) returns (GetJobStatusRes);
  rpc StopJob(StopJobReq) returns (StopJobRes);
  rpc ListJobs(EmptyRequest) returns (Jobs);
  rpc GetJobLogs(GetJobLogsReq) returns (stream DataChunk);
}
```

### Job Resource Specification

```protobuf
message RunJobReq {
  string command = 1;                    // Command to execute
  repeated string args = 2;              // Command arguments
  int32 maxCPU = 3;                     // CPU limit (percentage)
  int32 maxMemory = 4;                  // Memory limit (MB)
  int32 maxIOBPS = 5;                   // I/O limit (bytes/second)
}
```

### CLI Commands

```bash
# Job Management
cli run [--max-cpu=N] [--max-memory=N] [--max-iobps=N] <command> [args...]
cli get <job-id>          # Get detailed job information
cli list                  # List all jobs with status
cli stop <job-id>         # Stop running job
cli log <job-id>          # Stream real-time job output

# Connection Configuration
cli --server=host:port    # Connect to specific server
```

## 🎯 Performance

### System Capabilities

- **Concurrent Jobs**: Hundreds of simultaneous jobs with proper isolation
- **Resource Enforcement**: Hard limits via cgroups v2
- **Real-time Streaming**: Low-latency log streaming with buffering
- **Memory Efficiency**: Minimal overhead per job (~2MB per isolated job)
- **Network Performance**: Host networking for maximum throughput

### Resource Defaults

When resource limits are not specified:

- **CPU**: 100% (one full core)
- **Memory**: 512 MB
- **I/O**: Unlimited
- **Network**: Host networking (shared with system)

## 🤝 Contributing

We welcome contributions! Here's how to get started:

### Development Setup

```bash
# Clone and setup
git clone https://github.com/ehsaniara/job-worker.git
cd job-worker
make setup-dev

# Run tests
make test

# Test deployment
make deploy-quick
```

### Code Quality

- Follow `gofmt` and `golint` standards
- Comprehensive test coverage
- Update documentation for new features
- Ensure security best practices

## 📊 Monitoring

### Built-in Metrics

- Job execution statistics
- Resource usage tracking
- System health monitoring
- Performance metrics

### Integration

```bash
# Service status
systemctl status worker

# Live logs
journalctl -u worker -f

# Job metrics
./cli list  # Shows job counts and status
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [gRPC](https://grpc.io/) - High-performance RPC framework
- [Linux Cgroups v2](https://www.kernel.org/doc/Documentation/cgroup-v2.txt) - Resource isolation and management
- [Cobra](https://github.com/spf13/cobra) - Modern CLI framework for Go
- [Protocol Buffers](https://developers.google.com/protocol-buffers) - Language-neutral data serialization
- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html) - Process isolation mechanisms

---

<div align="center">
  <sub>by <a href="https://github.com/ehsaniara">Jay Ehsaniara</a></sub>
</div>