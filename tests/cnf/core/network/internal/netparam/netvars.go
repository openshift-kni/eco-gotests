package netparam

import (
	"time"

	"github.com/openshift-kni/eco-gotests/tests/cnf/core/internal/coreparams"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(coreparams.Labels, Label)
	// OperatorConfigDaemon defaults SR-IOV config daemon daemonset.
	OperatorConfigDaemon = "sriov-network-config-daemon"
	// OperatorDevicePlugin defaults SR-IOV device plugin daemonset.
	OperatorDevicePlugin = "sriov-device-plugin"
	// OperatorWebhook defaults SR-IOV webhook daemonset.
	OperatorWebhook = "operator-webhook"
	// OperatorResourceInjector defaults SR-IOV network resource injector daemonset.
	OperatorResourceInjector = "network-resources-injector"
	// OperatorSriovDaemonsets represents all default SR-IOV operator daemonset names.
	OperatorSriovDaemonsets = []string{OperatorConfigDaemon, OperatorWebhook, OperatorResourceInjector}
	// DefaultTimeout represents the default timeout for most of Eventually/PollImmediate functions.
	DefaultTimeout = 300 * time.Second
	// MCOWaitTimeout represent timeout for mco operations.
	MCOWaitTimeout = 60 * time.Minute
	// VtySh represents default vtysh cmd prefix.
	VtySh = []string{"vtysh", "-c"}
	// MlxDeviceID is the Mellanox SRIOV device ID.
	MlxDeviceID = "1017"
	// MlxBFDeviceID is the Mellanox Bluefield SRIOV Device ID.
	MlxBFDeviceID = "a2d6"
)
