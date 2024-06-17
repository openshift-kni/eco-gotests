package rdscorecommon

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/rbac"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
	v13 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createServiceAccount(saName, nsName string) {
	ginkgo.By(fmt.Sprintf("Creating ServiceAccount %q in %q namespace",
		saName, nsName))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating SA %q in %q namespace",
		saName, nsName)

	insureNamespaceExists(nsName)

	deploySa := serviceaccount.NewBuilder(rdscoreinittools.APIClient, saName, nsName)

	if !deploySa.Exists() {
		var ctx ginkgo.SpecContext

		deploySa, err := deploySa.Create()

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating SA %q in %q namespace: %v",
				saName, nsName, err)
		}

		gomega.Eventually(func() bool {
			if !deploySa.Exists() {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating SA %q in %q namespace",
					saName, nsName)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created SA %q in %q namespace",
				deploySa.Definition.Name, deploySa.Definition.Namespace)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(gomega.BeTrue(),
			fmt.Sprintf("Failed to create ServiceAccount %q in %q namespace", saName, nsName))
	}
}

func deleteServiceAccount(saName, nsName string) {
	ginkgo.By("Removing Service Account")
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Assert SA %q exists in %q namespace",
		saName, nsName)

	var ctx ginkgo.SpecContext

	if deploySa, err := serviceaccount.Pull(
		rdscoreinittools.APIClient, saName, nsName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ServiceAccount %q found in %q namespace",
			saName, nsName)
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting ServiceAccount %q in %q namespace",
			saName, nsName)

		err := deploySa.Delete()

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting ServiceAccount %q in %q namespace: %v",
				saName, nsName, err)
		}

		gomega.Eventually(func() bool {
			if deploySa.Exists() {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting ServiceAccount %q in %q namespace",
					saName, nsName)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted ServiceAccount %q in %q namespace",
				saName, nsName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(gomega.BeTrue(),
			fmt.Sprintf("Failed to delete ServiceAccount %q from %q ns", saName, nsName))
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ServiceAccount %q not found in %q namespace",
			saName, nsName)
	}
}

func deleteClusterRBAC(rbacName string) {
	ginkgo.By("Deleting Cluster RBAC")

	var ctx ginkgo.SpecContext

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Assert ClusterRoleBinding %q exists", rbacName)

	if crbSa, err := rbac.PullClusterRoleBinding(
		rdscoreinittools.APIClient,
		rbacName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ClusterRoleBinding %q found. Deleting...", rbacName)

		err := crbSa.Delete()

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting ClusterRoleBinding %q : %v",
				rbacName, err)
		}

		gomega.Eventually(func() bool {
			if crbSa.Exists() {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting ClusterRoleBinding %q",
					rbacName)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted ClusterRoleBinding %q", rbacName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(gomega.BeTrue(),
			"Failed to delete Cluster RBAC")
	}
}

//nolint:unparam
func createClusterRBAC(rbacName, clusterRole, saName, nsName string) {
	ginkgo.By("Creating RBAC for SA")

	var ctx ginkgo.SpecContext

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating ClusterRoleBinding %q", rbacName)

	insureNamespaceExists(nsName)

	crbSa := rbac.NewClusterRoleBindingBuilder(rdscoreinittools.APIClient,
		rbacName,
		clusterRole,
		v1.Subject{
			Name:      saName,
			Kind:      "ServiceAccount",
			Namespace: nsName,
		})

	crbSa, err := crbSa.Create()
	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Error Creating ClusterRoleBinding %q : %v", crbSa.Definition.Name, err)
	}

	gomega.Eventually(func() bool {
		if !crbSa.Exists() {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				"Error Creating ClusterRoleBinding %q : %v", crbSa.Definition.Name, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("ClusterRoleBinding %q created:\n\t%v",
			crbSa.Definition.Name, crbSa)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(gomega.BeTrue(),
		"Failed to create ClusterRoleBinding")
}

func deleteConfigMap(cmName, nsName string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Assert ConfigMap %q exists in %q namespace",
		cmName, nsName)

	if cmBuilder, err := configmap.Pull(
		rdscoreinittools.APIClient, cmName, nsName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("configMap %q found, deleting", cmName)

		var ctx ginkgo.SpecContext

		err := cmBuilder.Delete()

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting configMap %q : %v",
				cmName, err)
		}

		gomega.Eventually(func() bool {
			if cmBuilder.Exists() {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error deleting configMap %q", cmName)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleted configMap %q in %q namespace",
				cmName, nsName)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(gomega.BeTrue(),
			"Failed to delete configMap")
	}
}

func createConfigMap(cmName, nsName string, data map[string]string) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Create ConfigMap %q in %q namespace",
		cmName, nsName)

	insureNamespaceExists(nsName)

	cmBuilder := configmap.NewBuilder(rdscoreinittools.APIClient, cmName, nsName)
	cmBuilder.WithData(data)

	var ctx ginkgo.SpecContext

	cmResult, err := cmBuilder.Create()
	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating ConfigMap %q in %q namespace: %v",
			cmName, nsName, err)
	}

	gomega.Eventually(func() bool {

		if !cmResult.Exists() {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating ConfigMap %q in %q namespace",
				cmName, nsName)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created ConfigMap %q in %q namespace",
			cmResult.Definition.Name, nsName)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithPolling(1*time.Minute).Should(gomega.BeTrue(),
		"Failed to crete configMap")
}

func deleteDeployments(dName, nsName string) {
	ginkgo.By(fmt.Sprintf("Removing test deployment %q from %q ns", dName, nsName))

	if deploy, err := deployment.Pull(rdscoreinittools.APIClient, dName, nsName); err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting deployment %q from %q namespace",
			deploy.Definition.Name, nsName)

		deploy, err = deploy.WithReplicas(int32(0)).Update()
		gomega.Expect(err).ToNot(gomega.HaveOccurred(),
			fmt.Sprintf("failed to reduce deployment %q in namespace %q replicas count to zero due to %v",
				dName, nsName, err))

		err = deploy.DeleteAndWait(300 * time.Second)
		gomega.Expect(err).ToNot(gomega.HaveOccurred(),
			fmt.Sprintf("failed to delete deployment %q in namespace %q due to %v", dName, nsName, err))
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deployment %q not found in %q namespace",
			dName, nsName)
	}
}

func findPodWithSelector(fNamespace, podLabel string) []*pod.Builder {
	ginkgo.By(fmt.Sprintf("Getting pod(s) matching selector %q", podLabel))

	var (
		podMatchingSelector []*pod.Builder
		err                 error
		ctx                 ginkgo.SpecContext
	)

	podOneSelector := v12.ListOptions{
		LabelSelector: podLabel,
	}

	gomega.Eventually(func() bool {
		podMatchingSelector, err = pod.List(rdscoreinittools.APIClient, fNamespace, podOneSelector)
		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list pods in %q namespace: %v",
				fNamespace, err)

			return false
		}

		if len(podMatchingSelector) == 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found 0 pods matching label %q in namespace %q",
				podLabel, fNamespace)

			return false
		}

		return true
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(gomega.BeTrue(),
		fmt.Sprintf("Failed to find pod matching label %q in %q namespace", podLabel, fNamespace))

	return podMatchingSelector
}

func defineContainer(cName, cImage string, cCmd []string, cRequests, cLimits map[string]string) *pod.ContainerBuilder {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Creating container %q", cName)
	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	ginkgo.By("Defining SecurityContext")

	var trueFlag = true

	userUID := new(int64)

	*userUID = 0

	secContext := &v13.SecurityContext{
		RunAsUser:  userUID,
		Privileged: &trueFlag,
		SeccompProfile: &v13.SeccompProfile{
			Type: v13.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &v13.Capabilities{
			Add: []v13.Capability{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "IPC_LOCK"},
		},
	}

	ginkgo.By("Setting SecurityContext")

	deployContainer = deployContainer.WithSecurityContext(secContext)
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Container One definition: %#v", deployContainer)

	ginkgo.By("Dropping ALL security capability")

	deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)

	ginkgo.By("Adding VolumeMount to container")

	volMount := v13.VolumeMount{
		Name:      "configs",
		MountPath: "/opt/net/",
		ReadOnly:  false,
	}

	deployContainer = deployContainer.WithVolumeMount(volMount)

	if len(cRequests) != 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Processing container's requests")

		containerRequests := v13.ResourceList{}

		for key, val := range cRequests {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing container's request: %q - %q", key, val)

			containerRequests[v13.ResourceName(key)] = resource.MustParse(val)
		}

		deployContainer = deployContainer.WithCustomResourcesRequests(containerRequests)
	}

	if len(cLimits) != 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Processing container's limits")

		containerLimits := v13.ResourceList{}

		for key, val := range cLimits {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing container's limit: %q - %q", key, val)

			containerLimits[v13.ResourceName(key)] = resource.MustParse(val)
		}

		deployContainer = deployContainer.WithCustomResourcesLimits(containerLimits)
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("%q container's  definition:\n%#v", cName, deployContainer)

	return deployContainer
}

func verifyMsgInPodLogs(podObj *pod.Builder, msg, cName string, timeSpan time.Time) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Parsing duration %q", timeSpan)

	var (
		podLog string
		err    error
		ctx    ginkgo.SpecContext
	)

	gomega.Eventually(func() bool {
		logStartTimestamp := time.Since(timeSpan)
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\tTime duration is %s", logStartTimestamp)

		if logStartTimestamp.Abs().Seconds() < 1 {
			logStartTimestamp, err = time.ParseDuration("1s")
			gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to parse time duration")
		}

		podLog, err = podObj.GetLog(logStartTimestamp, cName)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to get logs from pod %q: %v", podObj.Definition.Name, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Logs from pod %s:\n%s", podObj.Definition.Name, podLog)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(gomega.BeTrue(),
		fmt.Sprintf("Failed to get logs from pod %q", podObj.Definition.Name))

	gomega.Expect(podLog).Should(gomega.ContainSubstring(msg))
}

func insureNamespaceExists(nsName string) {
	ginkgo.By(fmt.Sprintf("Insure namespace %q exists", nsName))

	createNs := namespace.NewBuilder(rdscoreinittools.APIClient, nsName)

	if !createNs.Exists() {
		var ctx ginkgo.SpecContext

		createNs, err := createNs.Create()

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating namespace %q: %v", nsName, err)
		}

		gomega.Eventually(func() bool {
			if !createNs.Exists() {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error creating namespace %q", nsName)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Created namespace %q", createNs.Definition.Name)

			return true
		}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(gomega.BeTrue(),
			fmt.Sprintf("Failed to create namespace %q", nsName))
	}
}
