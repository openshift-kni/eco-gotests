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

	// RhwaLogLevel represents the log level used in Glog statements.
	RhwaLogLevel = 90

	// RapidastTemplateFolder represents the relative route of the folder storing the needed templates.
	RapidastTemplateFolder = "../internal/dast/config-files"

	// RapidastTemplateFile represents the file name used for rapiDast configuration.
	RapidastTemplateFile = "rapidastConfig.yaml"

	// RapidastImage represents the rapidast container image used for dast testing.
	RapidastImage = "quay.io/redhatproductsecurity/rapidast:2.7.0"
)
