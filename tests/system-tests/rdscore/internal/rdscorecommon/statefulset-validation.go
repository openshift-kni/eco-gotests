package rdscorecommon

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	"github.com/openshift-kni/eco-goinfra/pkg/statefulset"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func defineHeadlessService(svcName, svcNSName string,
	svcSelector map[string]string, svcPort corev1.ServicePort) *service.Builder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining headless service %q in %q ns", svcName, svcNSName)

	svcBuilder := service.NewBuilder(APIClient, svcName, svcNSName, svcSelector, svcPort)

	// Make it headless by setting ClusterIP to None
	svcBuilder.Definition.Spec.ClusterIP = "None"

	return svcBuilder
}

func defineStatefulsetContainer(cName, cImage string, cCmd []string, cRequests, cLimits map[string]string) *pod.ContainerBuilder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining container %q with %q image and command %q",
		cName, cImage, cCmd)

	cBuilder := pod.NewContainerBuilder(cName, cImage, cCmd)

	if cRequests != nil {
		containerRequests := corev1.ResourceList{}

		for key, val := range cRequests {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing container's request: %q - %q", key, val)

			containerRequests[corev1.ResourceName(key)] = resource.MustParse(val)
		}

		cBuilder = cBuilder.WithCustomResourcesRequests(containerRequests)
	}

	if cLimits != nil {
		containerLimits := corev1.ResourceList{}

		for key, val := range cLimits {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing container's limit: %q - %q", key, val)

			containerLimits[corev1.ResourceName(key)] = resource.MustParse(val)
		}

		cBuilder = cBuilder.WithCustomResourcesLimits(containerLimits)
	}

	return cBuilder
}

func withRequiredLabelPodAffinity(
	st *statefulset.Builder,
	matchLabels map[string]string,
	ns []string,
	topologyKey string) error {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Adding required pod affinity to statefulset %q",
		st.Definition.Name)
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("PodAffinity 'matchLabels': %q", matchLabels)
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("PodAffinity 'namespaces': %q", ns)

	if matchLabels == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Option 'matchLabels' is not set")

		return fmt.Errorf("Option 'matchLabels' is not set")
	}

	if topologyKey == "" {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Option 'topologyKey' is not set")

		return fmt.Errorf("Option 'topologyKey' is not set")
	}

	if len(ns) == 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Option 'namespaces' is not set")

		return fmt.Errorf("Option 'namespaces' is not set")
	}

	st.Definition.Spec.Template.Spec.Affinity = &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: matchLabels,
					},
					TopologyKey: topologyKey,
					Namespaces:  ns,
				},
			},
		},
	}

	return nil
}
