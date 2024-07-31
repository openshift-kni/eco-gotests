package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite Suite name.
	LabelSuite = "diskencryption"
	// TimeoutWaitRegex timeout for a matching regex during boot.
	TimeoutWaitRegex = 10 * time.Minute
	// TimeoutClusterRecovery timeout for cluster recovery.
	TimeoutClusterRecovery = 10 * time.Minute
	// PollingIntervalBMC interval to poll the BMC after an error.
	PollingIntervalBMC = 30 * time.Second
	// TimeoutWaitingOnBMC timeout until failing a BMC command.
	TimeoutWaitingOnBMC = 10 * time.Minute
	// PollingIntervalK8S interval to poll cluster after an error.
	PollingIntervalK8S = 1 * time.Second
	// TimeoutWaitingOnK8S timeout until failing a cluster command.
	TimeoutWaitingOnK8S = 10 * time.Second
	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
	// RetryInterval retry interval for node exec commands.
	RetryInterval = 10 * time.Second
	// RetryCount retry count for node exec commands.
	RetryCount = 6
)
