package params

import "github.com/golang/glog"

// PrivilegedNSLabels represents privileged labels.
var (
	PrivilegedNSLabels = map[string]string{
		"pod-security.kubernetes.io/audit":               "privileged",
		"pod-security.kubernetes.io/enforce":             "privileged",
		"pod-security.kubernetes.io/warn":                "privileged",
		"security.openshift.io/scc.podSecurityLabelSync": "false",
	}
	// LogLevel is the verbosity for global tools.
	LogLevel glog.Level = 90
	// OpenshiftLoggingNamespace is the namespace where logging pods are.
	OpenshiftLoggingNamespace = "openshift-logging"
)
