package randutestworkload

import (
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-goinfra/pkg/statefulset"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsinittools"
)

// CleanNameSpace function removes all objects inside the namespace plus sriov networks whose
// NetworkNamespace spec matches the namespace.
func CleanNameSpace(cleanTimeout time.Duration, nsname string) error {
	err := namespace.NewBuilder(APIClient, nsname).
		CleanObjects(cleanTimeout, deployment.GetGVR(), statefulset.GetGVR())
	if err != nil {
		glog.V(100).Infof("Failed to clean up objects in namespace: %s", nsname)

		return err
	}

	err = namespace.NewBuilder(APIClient, nsname).DeleteAndWait(cleanTimeout)
	if err != nil {
		glog.V(100).Infof("Failed to remove namespace: %s", nsname)

		return err
	}

	return sriov.CleanAllNetworksByTargetNamespace(APIClient,
		SystemTestsTestConfig.SriovOperatorNamespace,
		nsname)
}
