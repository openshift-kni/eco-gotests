package systemtestsparams

var (
	// HardRebootDeploymentName represents the deployment name used in hard reboot test.
	HardRebootDeploymentName = "ipmitoool-hardreboot"
	// PrivilegedNSLabels represents privileged labels.
	PrivilegedNSLabels = map[string]string{
		"pod-security.kubernetes.io/audit":               "privileged",
		"pod-security.kubernetes.io/enforce":             "privileged",
		"pod-security.kubernetes.io/warn":                "privileged",
		"security.openshift.io/scc.podSecurityLabelSync": "false",
	}
)
