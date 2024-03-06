package spkcommon

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/url"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/spk/internal/spkinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/spk/internal/spkparams"
)

func reachURL(targetURL string, expectedCode int) {
	glog.V(spkparams.SPKLogLevel).Infof("Accessing %q via SPK Ingress", targetURL)

	var ctx SpecContext

	Eventually(func() bool {
		data, httpCode, err := url.Fetch(targetURL, "GET")

		if err != nil {
			glog.V(spkparams.SPKLogLevel).Infof("Failed to reach %q: %v", targetURL, err)

			return false
		}

		glog.V(spkparams.SPKLogLevel).Infof("Reached URL %q", targetURL)
		glog.V(spkparams.SPKLogLevel).Infof("HTTP Code: %d", httpCode)
		glog.V(spkparams.SPKLogLevel).Infof("Reached data\n%v", data)

		return httpCode == expectedCode
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Failed to reach %q URL", targetURL))
}

// AssertIPv4WorkloadURL access workload via IPv4 address.
func AssertIPv4WorkloadURL(ctx SpecContext) {
	glog.V(spkparams.SPKLogLevel).Infof("*** AssertIPv4WorkloadURL ***")

	if SPKConfig.IngressTCPIPv4URL == "" {
		Skip("IPv4 URL for SPK backed workload not defined")
	}

	reachURL(SPKConfig.IngressTCPIPv4URL, int(200))
}

// AssertIPv6WorkloadURL access workload via IPv6 address.
func AssertIPv6WorkloadURL(ctx SpecContext) {
	glog.V(spkparams.SPKLogLevel).Infof("*** AssertIPv6WorkloadURL ***")

	if SPKConfig.IngressTCPIPv6URL == "" {
		Skip("IPv6 URL for SPK backed workload not defined")
	}

	reachURL(SPKConfig.IngressTCPIPv6URL, int(200))
}
