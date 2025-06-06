package config

import "time"

const (

	// DefaultMemoryLimitMB Default memory limit in MB (1MB)
	DefaultMemoryLimitMB = int32(1)
	// DefaultCPULimitPercent Default CPU limit (10% of one CPU)
	DefaultCPULimitPercent = int32(10)
	// DefaultIOBPS Default IO Byte Per Second
	DefaultIOBPS = int32(0)

	// CleanupTimeout timeout for the cleanup operation
	CleanupTimeout = 5 * time.Second

	MaxRecvMsgSize    = 512 * 1024      // 512KB
	MaxSendMsgSize    = 4 * 1024 * 1024 // 4MB
	MaxHeaderListSize = 1 * 1024 * 1024 // 1MB
)
