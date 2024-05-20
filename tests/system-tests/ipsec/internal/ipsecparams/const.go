package ipsecparams

import (
	"time"
)

const (
	// Label represents IPSec system tests label that can be used for test cases selection.
	Label = "ipsec"
	// DefaultTimeout is the timeout used for test resources creation.
	DefaultTimeout = 900 * time.Second
	// IpsecLogLevel configures logging level for IPSec related tests.
	IpsecLogLevel = 90
)
