package os

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

import (
	"os"
	"syscall"
)

//counterfeiter:generate . SyscallInterface
type SyscallInterface interface {
	// Kill sends a signal to a process or process group
	// - Positive pid: kills the specific process
	// - Negative pid: kills the process group (all processes in the group)
	// This follows the standard Unix convention
	Kill(pid int, sig syscall.Signal) error
	CreateProcessGroup() *syscall.SysProcAttr
	Exec(string, []string, []string) error
	Mount(source string, target string, fstype string, flags uintptr, data string) error
	Unmount(target string, flags int) error
}

//counterfeiter:generate . OsInterface
type OsInterface interface {
	WriteFile(name string, data []byte, perm os.FileMode) error
	Executable() (string, error)
	Stat(name string) (os.FileInfo, error)
	Environ() []string
	Getenv(key string) string
	Getpid() int
	ReadFile(path string) ([]byte, error)
}

//counterfeiter:generate . CommandFactory
type CommandFactory interface {
	CreateCommand(name string, args ...string) Command
}

//counterfeiter:generate . Command
type Command interface {
	Start() error
	Wait() error
	Process() Process
	SetStdout(w interface{})
	SetStderr(w interface{})
	SetSysProcAttr(attr *syscall.SysProcAttr)
	SetEnv([]string)
}

//counterfeiter:generate . Process
type Process interface {
	Pid() int
	Kill() error
}

//counterfeiter:generate . ExecInterface
type ExecInterface interface {
	LookPath(file string) (string, error)
}
