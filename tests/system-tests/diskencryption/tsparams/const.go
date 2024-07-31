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
	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
