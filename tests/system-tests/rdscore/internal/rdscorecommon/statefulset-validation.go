package rdscorecommon

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"

	"github.com/openshift-kni/eco-goinfra/pkg/statefulset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WaitAllStatefulsetsReady checks all statefulset objects are in Ready state.
func WaitAllStatefulsetsReady(ctx SpecContext) {
	By("Checking all statefulsets are Ready")

	Eventually(func() bool {
		var (
			statefulsetList []*statefulset.Builder
			err             error
			notReadySets    int
		)

		statefulsetList, err = statefulset.ListInAllNamespaces(APIClient, metav1.ListOptions{})

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list statefulsets in all namespaces: %s", err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d statefulsets", len(statefulsetList))

		notReadySets = 0

		for _, sfSet := range statefulsetList {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				"Processing statefulset %q in %q namespace", sfSet.Definition.Name, sfSet.Definition.Namespace)

			if sfSet.Object.Status.ReadyReplicas != *sfSet.Definition.Spec.Replicas {
				msg := fmt.Sprintf("Statefulset %q in %q namespace has %d ready replicas but expects %d",
					sfSet.Definition.Name, sfSet.Definition.Namespace,
					sfSet.Object.Status.ReadyReplicas, *sfSet.Definition.Spec.Replicas)

				glog.V(rdscoreparams.RDSCoreLogLevel).Infof(msg)

				notReadySets++
			} else {
				msg := fmt.Sprintf("Statefulset %q in %q namespace has %d ready replicas and expects %d",
					sfSet.Definition.Name, sfSet.Definition.Namespace,
					sfSet.Object.Status.ReadyReplicas, *sfSet.Definition.Spec.Replicas)

				glog.V(rdscoreparams.RDSCoreLogLevel).Infof(msg)
			}
		}

		return notReadySets == 0
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(15*time.Minute).Should(BeTrue(),
		"There are not Ready statefulsets all namespaces")
}
