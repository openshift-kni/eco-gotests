package ipsec_system_test

import (
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ipsec/internal/ipsecinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ipsec/internal/ipsecparams"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ipsec/internal/ipsectunnel"
)

var _ = Describe(
	"IpsecTunnelAtDeployment",
	Label("IpsecTunnelAtDeployment"),
	Ordered,
	ContinueOnFailure,
	func() {
		It("Asserts the IPSec tunnel connected successfully at OCP deployment", func() {
			glog.V(ipsecparams.IpsecLogLevel).Infof("Check IPSec tunnel connection at OCP deployment")

			nodeNames, _ := GetNodeNames()
			for _, nodeName := range nodeNames {
				glog.V(ipsecparams.IpsecLogLevel).Infof("Check IPSec tunnel connection on node %s", nodeName)
				err := ipsectunnel.TunnelConnected(nodeName)
				Expect(err).ToNot(HaveOccurred(), "Error: The IPSec tunnel is not connected.")
			}
		})
	},
)
