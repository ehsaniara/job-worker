//go:build linux

package process

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"syscall"
	"time"

	"worker/pkg/logger"
	osinterface "worker/pkg/os"
)

const (
	ProcessStartTimeout = 10 * time.Second
)

// Launcher handles process launching with namespace support
type Launcher struct {
	cmdFactory  osinterface.CommandFactory
	syscall     osinterface.SyscallInterface
	osInterface osinterface.OsInterface
	validator   *Validator
	logger      *logger.Logger
}

// LaunchConfig contains all configuration for launching a process
type LaunchConfig struct {
	InitPath      string
	Environment   []string
	SysProcAttr   *syscall.SysProcAttr
	Stdout        io.Writer
	Stderr        io.Writer
	NamespacePath string
	NamespaceType string
	NeedsNSJoin   bool
	JobID         string
	Command       string
	Args          []string
}

// LaunchResult contains the result of a process launch
type LaunchResult struct {
	PID     int32
	Command osinterface.Command
	Error   error
}

// NewLauncher creates a new process launcher
func NewLauncher(
	cmdFactory osinterface.CommandFactory,
	syscall osinterface.SyscallInterface,
	osInterface osinterface.OsInterface,
	validator *Validator,
) *Launcher {
	return &Launcher{
		cmdFactory:  cmdFactory,
		syscall:     syscall,
		osInterface: osInterface,
		validator:   validator,
		logger:      logger.New().WithField("component", "process-launcher"),
	}
}

// LaunchProcess starts process with namespace isolation and proper cleanup on failure
func (l *Launcher) LaunchProcess(ctx context.Context, config *LaunchConfig) (*LaunchResult, error) {
	if config == nil {
		return nil, fmt.Errorf("launch config cannot be nil")
	}

	log := l.logger.WithFields("jobID", config.JobID, "command", config.Command)
	log.Info("launching process",
		"needsNSJoin", config.NeedsNSJoin,
		"namespacePath", config.NamespacePath,
		"namespaceType", config.NamespaceType)

	// Validate configuration
	if err := l.validateLaunchConfig(config); err != nil {
		return nil, fmt.Errorf("invalid launch config: %w", err)
	}

	// Use pre-fork namespace setup approach for namespace joining
	resultChan := make(chan *LaunchResult, 1)

	go l.launchInGoroutine(config, resultChan)

	// Wait with timeout to prevent hanging on process start failures
	select {
	case result := <-resultChan:
		if result.Error != nil {
			log.Error("failed to start process in goroutine", "error", result.Error)
			return nil, fmt.Errorf("failed to start process: %w", result.Error)
		}

		log.Info("process started successfully", "pid", result.PID)
		return result, nil

	case <-ctx.Done():
		log.Warn("context cancelled while starting process")
		return nil, ctx.Err()

	case <-time.After(ProcessStartTimeout):
		log.Error("timeout waiting for process to start")
		return nil, fmt.Errorf("timeout waiting for process to start")
	}
}

// launchInGoroutine launches the process in a separate goroutine with proper namespace handling
func (l *Launcher) launchInGoroutine(config *LaunchConfig, resultChan chan<- *LaunchResult) {
	defer func() {
		if r := recover(); r != nil {
			resultChan <- &LaunchResult{
				Error: fmt.Errorf("panic in launch goroutine: %v", r),
			}
		}
	}()

	log := l.logger.WithField("jobID", config.JobID)

	// Lock this goroutine to the OS thread for namespace operations
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Join namespace if needed (before forking) - now supports any namespace type
	if config.NeedsNSJoin && config.NamespacePath != "" {
		log.Debug("joining namespace before fork",
			"nsPath", config.NamespacePath,
			"nsType", config.NamespaceType)

		if err := l.joinNamespace(config.NamespacePath, config.NamespaceType); err != nil {
			resultChan <- &LaunchResult{
				Error: fmt.Errorf("failed to join namespace: %w", err),
			}
			return
		}
		log.Debug("successfully joined namespace", "nsType", config.NamespaceType)
	}

	// Start the process (which will inherit the current namespace)
	startTime := time.Now()
	cmd, err := l.createAndStartCommand(config)
	if err != nil {
		resultChan <- &LaunchResult{
			Error: fmt.Errorf("failed to start command: %w", err),
		}
		return
	}

	process := cmd.Process()
	if process == nil {
		resultChan <- &LaunchResult{
			Error: fmt.Errorf("process is nil after start"),
		}
		return
	}

	duration := time.Since(startTime)
	log.Debug("process started in goroutine", "pid", process.Pid(), "duration", duration)

	resultChan <- &LaunchResult{
		PID:     int32(process.Pid()),
		Command: cmd,
		Error:   nil,
	}
}

// Generic namespace joining that supports multiple namespace types
func (l *Launcher) joinNamespace(nsPath string, nsType string) error {
	if nsPath == "" {
		return fmt.Errorf("namespace path cannot be empty")
	}

	if nsType == "" {
		return fmt.Errorf("namespace type cannot be empty")
	}

	// Check if namespace file exists
	if _, err := l.osInterface.Stat(nsPath); err != nil {
		return fmt.Errorf("namespace file does not exist: %s (%w)", nsPath, err)
	}

	l.logger.Debug("opening namespace file", "nsPath", nsPath, "nsType", nsType)

	// Open the namespace file
	fd, err := syscall.Open(nsPath, syscall.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open namespace file %s: %w", nsPath, err)
	}
	defer func() {
		if closeErr := syscall.Close(fd); closeErr != nil {
			l.logger.Warn("failed to close namespace file descriptor", "error", closeErr)
		}
	}()

	// Map namespace type to appropriate clone flag
	var cloneFlag uintptr
	switch nsType {
	case "mount":
		cloneFlag = syscall.CLONE_NEWNS
	case "network":
		cloneFlag = syscall.CLONE_NEWNET
	case "pid":
		cloneFlag = syscall.CLONE_NEWPID
	case "ipc":
		cloneFlag = syscall.CLONE_NEWIPC
	case "uts":
		cloneFlag = syscall.CLONE_NEWUTS
	case "user":
		cloneFlag = syscall.CLONE_NEWUSER
	case "cgroup":
		cloneFlag = syscall.CLONE_NEWCGROUP
	default:
		return fmt.Errorf("unsupported namespace type: %s", nsType)
	}

	l.logger.Debug("calling setns syscall", "fd", fd, "nsPath", nsPath, "nsType", nsType, "cloneFlag", cloneFlag)

	// Call setns system call (x86_64 syscall number for setns)
	const SysSetnsX86_64 = 308
	_, _, errno := syscall.Syscall(SysSetnsX86_64, uintptr(fd), cloneFlag, 0)
	if errno != 0 {
		return fmt.Errorf("setns syscall failed for %s namespace %s: %v", nsType, nsPath, errno)
	}

	l.logger.Debug("successfully joined namespace", "nsPath", nsPath, "nsType", nsType)
	return nil
}

// createAndStartCommand creates and starts the command with proper configuration
func (l *Launcher) createAndStartCommand(config *LaunchConfig) (osinterface.Command, error) {
	// Create command
	cmd := l.cmdFactory.CreateCommand(config.InitPath)

	// Set environment
	if config.Environment != nil {
		cmd.SetEnv(config.Environment)
	}

	// Set stdout/stderr
	if config.Stdout != nil {
		cmd.SetStdout(config.Stdout)
	}
	if config.Stderr != nil {
		cmd.SetStderr(config.Stderr)
	}

	// Set system process attributes (namespaces, process group, etc.)
	if config.SysProcAttr != nil {
		cmd.SetSysProcAttr(config.SysProcAttr)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// validateLaunchConfig validates the launch configuration
func (l *Launcher) validateLaunchConfig(config *LaunchConfig) error {
	if config.InitPath == "" {
		return fmt.Errorf("init path cannot be empty")
	}

	if config.JobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	// Validate init path exists and is executable
	if err := l.validator.ValidateInitPath(config.InitPath); err != nil {
		return fmt.Errorf("invalid init path: %w", err)
	}

	// Validate environment if provided
	if config.Environment != nil {
		if err := l.validator.ValidateEnvironment(config.Environment); err != nil {
			return fmt.Errorf("invalid environment: %w", err)
		}
	}

	// Validate namespace configuration if namespace join is needed
	if config.NeedsNSJoin {
		if config.NamespacePath == "" {
			return fmt.Errorf("namespace path required when NeedsNSJoin is true")
		}

		if config.NamespaceType == "" {
			return fmt.Errorf("namespace type required when NeedsNSJoin is true")
		}

		// Validate that the namespace type is supported
		supportedTypes := []string{"mount", "network", "pid", "ipc", "uts", "user", "cgroup"}
		isSupported := false
		for _, supportedType := range supportedTypes {
			if config.NamespaceType == supportedType {
				isSupported = true
				break
			}
		}
		if !isSupported {
			return fmt.Errorf("unsupported namespace type: %s", config.NamespaceType)
		}

		// Check if namespace file exists
		if _, err := l.osInterface.Stat(config.NamespacePath); err != nil {
			return fmt.Errorf("namespace file validation failed: %w", err)
		}
	}

	return nil
}
