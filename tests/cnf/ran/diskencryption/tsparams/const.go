package tsparams

import "time"

const (
	// LabelSuite Suite name.
	LabelSuite = "diskencryption"
	// TimeoutWaitRegex timeout for a matching regex during boot.
	TimeoutWaitRegex = 10 * time.Minute
	// TimeoutClusterRecovery timeout for cluster recovery.
	TimeoutClusterRecovery = 10 * time.Minute
	// PoolingIntervalBMC interval to pool the BMC after an error.
	PoolingIntervalBMC = 30 * time.Second
	// TimeoutWaitingOnBMC timeout until failing a BMC command.
	TimeoutWaitingOnBMC = 10 * time.Minute
)
