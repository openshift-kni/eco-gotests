package rhwaparams

import (
	"time"
)

const (
	// Label represents rhwa label that can be used for test cases selection.
	Label = "rhwa"
	// RhwaOperatorNs custom namespace of rhwa operators.
	RhwaOperatorNs = "openshift-workload-availability"
	// DefaultTimeout represents the default timeout.
	DefaultTimeout = 300 * time.Second

	RhwaLogLevel = 90

	RapidastTemplateFolder = "../internal/dast/config-files"
	RapidastTemplateFile   = "rapidastConfig.yaml"
	RapidastImage          = "quay.io/redhatproductsecurity/rapidast:2.7.0"
)
