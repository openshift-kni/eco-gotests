package tsparams

const (
	// LabelSuite represents upgrade label that can be used for test cases selection.
	LabelSuite = "upgrade"
	// LabelEndToEndUpgrade represents e2e label that can be used for test cases selection.
	LabelEndToEndUpgrade = "e2e"
	// LabelPrepAbortFlow represents prep-abort label that can be used for test cases selection.
	LabelPrepAbortFlow = "ibu-prep-abort"

	// IbuCguNamespace is the namespace where IBU CGUs created on target hub.
	IbuCguNamespace = "default"

	// PrePrepCguName is the name of pre-prep cgu.
	PrePrepCguName = "cgu-ibu-pre-prep"
	// PrePrepPolicyName is the name of managed policy used to create oadp configmap.
	PrePrepPolicyName = "group-ibu-oadp-cm-policy"

	// PrepCguName is the name of prep cgu.
	PrepCguName = "cgu-ibu-prep"
	// PrepPolicyName is the name of managed policy for ibu prep stage validation.
	PrepPolicyName = "group-ibu-prep-stage-policy"

	// UpgradeCguName is the name of upgrade cgu.
	UpgradeCguName = "cgu-ibu-upgrade"
	// UpgradePolicyName is the name of managed policy for ibu upgrade stage validation.
	UpgradePolicyName = "group-ibu-upgrade-stage-policy"

	// FinalizeCguName is the name of finalize cgu.
	FinalizeCguName = "cgu-ibu-finalize"
	// FinalizePolicyName is the name of managed policy for ibu idle stage validation.
	FinalizePolicyName = "group-ibu-finalize-stage-policy"
)
