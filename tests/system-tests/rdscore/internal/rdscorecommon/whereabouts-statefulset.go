package rdscorecommon

import (
	"fmt"
	"strconv"
	"strings"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateStatefulsetOnSameNode creates a statefulset on the same node.
//
//nolint:funlen
func CreateStatefulsetOnSameNode(ctx SpecContext) {
	const (
		myHeadlessSvcOne            = "rds-st-one-headless-1"
		myStatefulsetOne            = "rds-st-one"
		myStatefulsetOneLabel       = "app=rds-st-one"
		myStatefulsetOneReplicas    = 2
		myStatefulsetOneSA          = "rds-st-one-sa"
		myStatefulsetOneRBACRole    = "system:openshift:scc:nonroot-v2"
		myStatefulsetOneTopologyKey = "kubernetes.io/hostname"
	)

	// check if service exists and delete it if it does
	By(fmt.Sprintf("Checking that service %q doesn't exist in %q namespace",
		myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS))

	svcOne, err := service.Pull(APIClient, myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to get service %q in %q namespace: %s",
			myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS, err)
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting service %q in %q namespace",
			myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS)

		delError := svcOne.Delete()

		Expect(delError).ToNot(HaveOccurred(), "Failed to delete service %q in %q namespace",
			myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS)
	}

	// check if statefulset exists and delete it if it does
	By(fmt.Sprintf("Checking that statefulset %q doesn't exist in %q namespace",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS))

	stOne, err := statefulset.Pull(APIClient, myStatefulsetOne, RDSCoreConfig.WhereaboutNS)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to get statefulset %q in %q namespace: %s",
			myStatefulsetOne, RDSCoreConfig.WhereaboutNS, err)
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting statefulset %q in %q namespace",
			myStatefulsetOne, RDSCoreConfig.WhereaboutNS)

		delError := stOne.Delete()

		Expect(delError).ToNot(HaveOccurred(), "Failed to delete statefulset %q in %q namespace",
			myStatefulsetOne, RDSCoreConfig.WhereaboutNS)

		// wait for pods to be deleted
		By(fmt.Sprintf("Waiting for pods from %q statefulset in %q namespace to be deleted",
			myStatefulsetOne, RDSCoreConfig.WhereaboutNS))

		Eventually(func() bool {
			pods, err := pod.List(APIClient, RDSCoreConfig.WhereaboutNS, metav1.ListOptions{
				LabelSelector: myStatefulsetOneLabel,
			})

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list pods from %q statefulset in %q namespace: %s",
					myStatefulsetOne, RDSCoreConfig.WhereaboutNS, err)

				return false
			}

			return len(pods) == 0
		}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
			"Pods from %q statefulset in %q namespace are not deleted", myStatefulsetOne, RDSCoreConfig.WhereaboutNS)
	}

	// create a headless service
	By(fmt.Sprintf("Creating headless service %q in %q namespace",
		myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining headless service selector")

	svcLabelsMap := map[string]string{
		strings.Split(myStatefulsetOneLabel, "=")[0]: strings.Split(myStatefulsetOneLabel, "=")[1],
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining headless service port")

	parsedPort, err := strconv.Atoi(RDSCoreConfig.WhereaboutsSTOnePort)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to parse port number: %v", RDSCoreConfig.WhereaboutsSTOnePort))

	svcPort := corev1.ServicePort{
		Port:     int32(parsedPort),
		Protocol: corev1.ProtocolTCP,
	}

	svcOne = defineHeadlessService(myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS,
		svcLabelsMap, svcPort)

	By("Setting ipFamilyPolicy to 'RequireDualStack'")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Setting ipFamilyPolicy to 'RequireDualStack'")

	svcOne = svcOne.WithIPFamily([]corev1.IPFamily{"IPv4", "IPv6"},
		corev1.IPFamilyPolicyRequireDualStack)

	Eventually(func() error {
		svcOne, err = svcOne.Create()

		return err
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(1*time.Minute).Should(Succeed(),
		"Failed to create headless service %q in %q namespace", myHeadlessSvcOne, RDSCoreConfig.WhereaboutNS)

	// define statefulset container
	By("Defining statefulset container")

	stContainer := defineStatefulsetContainer("my-st-container", RDSCoreConfig.WhereaboutsSTImageOne,
		RDSCoreConfig.WhereaboutsSTOneCMD, nil, nil)

	Expect(stContainer).ToNot(BeNil(), "Failed to define statefulset container")

	By("Getting statefulset container config")

	stContainerCfg, err := stContainer.GetContainerCfg()

	Expect(err).ToNot(HaveOccurred(), "Failed to get statefulset container config")

	// define statefulset
	By("Defining statefulset")

	stOne = statefulset.NewBuilder(APIClient, myStatefulsetOne, RDSCoreConfig.WhereaboutNS, svcLabelsMap, stContainerCfg)

	nadMap := map[string]string{
		"k8s.v1.cni.cncf.io/networks": RDSCoreConfig.WhereaboutsSTOneNAD,
	}

	By(fmt.Sprintf("Adding pod annotations to statefulset %q in %q namespace",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS))

	err = withPodAnnotations(stOne, nadMap)

	Expect(err).ToNot(HaveOccurred(), "Failed to add pod annotations to statefulset %q in %q namespace",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Adding pod affinity to statefulset %q in %q namespace",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS))

	err = withRequiredLabelPodAffinity(stOne,
		svcLabelsMap,
		[]string{RDSCoreConfig.WhereaboutNS},
		myStatefulsetOneTopologyKey)

	Expect(err).ToNot(HaveOccurred(), "Failed to add required label pod affinity to statefulset %q in %q namespace",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Setting statefulset %q in %q namespace to %d replica",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS, myStatefulsetOneReplicas))

	replicas := int32(myStatefulsetOneReplicas)

	stOne.Definition.Spec.Replicas = &replicas

	By(fmt.Sprintf("Deleting service account %q in %q namespace",
		myStatefulsetOneSA, RDSCoreConfig.WhereaboutNS))

	deleteServiceAccount(myStatefulsetOneSA, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Creating service account %q in %q namespace",
		myStatefulsetOneSA, RDSCoreConfig.WhereaboutNS))

	createServiceAccount(myStatefulsetOneSA, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Deleting cluster role binding for service account %q in %q namespace",
		myStatefulsetOneSA, RDSCoreConfig.WhereaboutNS))

	deleteClusterRBAC(myStatefulsetOneRBACRole)

	By(fmt.Sprintf("Creating cluster role binding for service account %q in %q namespace",
		myStatefulsetOneSA, RDSCoreConfig.WhereaboutNS))

	createClusterRBAC(myStatefulsetOneSA, myStatefulsetOneRBACRole, myStatefulsetOneSA, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Setting statefulset %q in %q namespace to use service account %q",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS, myStatefulsetOneSA))

	stOne.Definition.Spec.Template.Spec.ServiceAccountName = myStatefulsetOneSA

	By(fmt.Sprintf("Creating statefulset %q in %q namespace", myStatefulsetOne, RDSCoreConfig.WhereaboutNS))

	Eventually(func() bool {
		stOne, err = stOne.Create()

		return err == nil
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		"Failed to create statefulset %q in %q namespace", myStatefulsetOne, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Waiting for statefulset %q in %q namespace to be ready",
		myStatefulsetOne, RDSCoreConfig.WhereaboutNS))

	Eventually(func() bool {
		exists := stOne.Exists()

		if exists {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Statefulset ReadyReplicas: %d",
				stOne.Object.Status.ReadyReplicas)

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Statefulset Spec.Replicas: %d",
				*stOne.Definition.Spec.Replicas)

			if stOne.Definition.Spec.Replicas != nil {
				return stOne.Object.Status.ReadyReplicas == *stOne.Definition.Spec.Replicas
			}

			return stOne.Object.Status.ReadyReplicas != 0
		}

		return false
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		"Statefulset %q in %q namespace is not ready", myStatefulsetOne, RDSCoreConfig.WhereaboutNS)
}

// CreateStatefulsetOnDifferentNode creates a statefulset on the different node.
//
//nolint:funlen
func CreateStatefulsetOnDifferentNode(ctx SpecContext) {
	const (
		myHeadlessSvcTwo            = "rds-st-two-headless-2"
		myStatefulsetTwo            = "rds-st-two"
		myStatefulsetTwoLabel       = "app=rds-st-two"
		myStatefulsetTwoReplicas    = 2
		myStatefulsetTwoSA          = "rds-st-two-sa"
		myStatefulsetTwoRBACRole    = "system:openshift:scc:nonroot-v2"
		myStatefulsetTwoTopologyKey = "kubernetes.io/hostname"
	)

	// check if service exists and delete it if it does
	By(fmt.Sprintf("Checking that service %q doesn't exist in %q namespace",
		myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS))

	svcOne, err := service.Pull(APIClient, myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to get service %q in %q namespace: %s",
			myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS, err)
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting service %q in %q namespace",
			myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS)

		delError := svcOne.Delete()

		Expect(delError).ToNot(HaveOccurred(), "Failed to delete service %q in %q namespace",
			myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS)
	}

	// check if statefulset exists and delete it if it does
	By(fmt.Sprintf("Checking that statefulset %q doesn't exist in %q namespace",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS))

	stOne, err := statefulset.Pull(APIClient, myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to get statefulset %q in %q namespace: %s",
			myStatefulsetTwo, RDSCoreConfig.WhereaboutNS, err)
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting statefulset %q in %q namespace",
			myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)

		delError := stOne.Delete()

		Expect(delError).ToNot(HaveOccurred(), "Failed to delete statefulset %q in %q namespace",
			myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)

		// wait for pods to be deleted
		By(fmt.Sprintf("Waiting for pods from %q statefulset in %q namespace to be deleted",
			myStatefulsetTwo, RDSCoreConfig.WhereaboutNS))

		Eventually(func() bool {
			pods, err := pod.List(APIClient, RDSCoreConfig.WhereaboutNS, metav1.ListOptions{
				LabelSelector: myStatefulsetTwoLabel,
			})

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to list pods from %q statefulset in %q namespace: %s",
					myStatefulsetTwo, RDSCoreConfig.WhereaboutNS, err)

				return false
			}

			return len(pods) == 0
		}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
			"Pods from %q statefulset in %q namespace are not deleted", myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)
	}

	// create a headless service
	By(fmt.Sprintf("Creating headless service %q in %q namespace",
		myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining headless service selector")

	svcLabelsMap := map[string]string{
		strings.Split(myStatefulsetTwoLabel, "=")[0]: strings.Split(myStatefulsetTwoLabel, "=")[1],
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Defining headless service port")

	parsedPort, err := strconv.Atoi(RDSCoreConfig.WhereaboutsSTTwoPort)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to parse port number: %v", RDSCoreConfig.WhereaboutsSTTwoPort))

	svcPort := corev1.ServicePort{
		Port:     int32(parsedPort),
		Protocol: corev1.ProtocolTCP,
	}

	svcOne = defineHeadlessService(myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS,
		svcLabelsMap, svcPort)

	By("Setting ipFamilyPolicy to 'RequireDualStack'")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Setting ipFamilyPolicy to 'RequireDualStack'")

	svcOne = svcOne.WithIPFamily([]corev1.IPFamily{"IPv4", "IPv6"},
		corev1.IPFamilyPolicyRequireDualStack)

	Eventually(func() error {
		svcOne, err = svcOne.Create()

		return err
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(1*time.Minute).Should(Succeed(),
		"Failed to create headless service %q in %q namespace", myHeadlessSvcTwo, RDSCoreConfig.WhereaboutNS)

	// define statefulset container
	By("Defining statefulset container")

	stContainer := defineStatefulsetContainer("my-st-container", RDSCoreConfig.WhereaboutsSTImageTwo,
		RDSCoreConfig.WhereaboutsSTTwoCMD, nil, nil)

	Expect(stContainer).ToNot(BeNil(), "Failed to define statefulset container")

	By("Getting statefulset container config")

	stContainerCfg, err := stContainer.GetContainerCfg()

	Expect(err).ToNot(HaveOccurred(), "Failed to get statefulset container config")

	// define statefulset
	By("Defining statefulset")

	stOne = statefulset.NewBuilder(APIClient, myStatefulsetTwo, RDSCoreConfig.WhereaboutNS, svcLabelsMap, stContainerCfg)

	nadMap := map[string]string{
		"k8s.v1.cni.cncf.io/networks": RDSCoreConfig.WhereaboutsSTTwoNAD,
	}

	By(fmt.Sprintf("Adding pod annotations to statefulset %q in %q namespace",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS))

	err = withPodAnnotations(stOne, nadMap)

	Expect(err).ToNot(HaveOccurred(), "Failed to add pod annotations to statefulset %q in %q namespace",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Adding pod affinity to statefulset %q in %q namespace",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS))

	err = withRequiredLabelPodAntiAffinity(stOne,
		svcLabelsMap,
		[]string{RDSCoreConfig.WhereaboutNS},
		myStatefulsetTwoTopologyKey)

	Expect(err).ToNot(HaveOccurred(), "Failed to add required label pod affinity to statefulset %q in %q namespace",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Setting statefulset %q in %q namespace to %d replica",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS, myStatefulsetTwoReplicas))

	replicas := int32(myStatefulsetTwoReplicas)

	stOne.Definition.Spec.Replicas = &replicas

	By(fmt.Sprintf("Deleting service account %q in %q namespace",
		myStatefulsetTwoSA, RDSCoreConfig.WhereaboutNS))

	deleteServiceAccount(myStatefulsetTwoSA, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Creating service account %q in %q namespace",
		myStatefulsetTwoSA, RDSCoreConfig.WhereaboutNS))

	createServiceAccount(myStatefulsetTwoSA, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Deleting cluster role binding for service account %q in %q namespace",
		myStatefulsetTwoSA, RDSCoreConfig.WhereaboutNS))

	deleteClusterRBAC(myStatefulsetTwoRBACRole)

	By(fmt.Sprintf("Creating cluster role binding for service account %q in %q namespace",
		myStatefulsetTwoSA, RDSCoreConfig.WhereaboutNS))

	createClusterRBAC(myStatefulsetTwoSA, myStatefulsetTwoRBACRole, myStatefulsetTwoSA, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Setting statefulset %q in %q namespace to use service account %q",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS, myStatefulsetTwoSA))

	stOne.Definition.Spec.Template.Spec.ServiceAccountName = myStatefulsetTwoSA

	By(fmt.Sprintf("Creating statefulset %q in %q namespace", myStatefulsetTwo, RDSCoreConfig.WhereaboutNS))

	Eventually(func() bool {
		stOne, err = stOne.Create()

		return err == nil
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		"Failed to create statefulset %q in %q namespace", myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)

	By(fmt.Sprintf("Waiting for statefulset %q in %q namespace to be ready",
		myStatefulsetTwo, RDSCoreConfig.WhereaboutNS))

	Eventually(func() bool {
		exists := stOne.Exists()

		if exists {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Statefulset ReadyReplicas: %d",
				stOne.Object.Status.ReadyReplicas)

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Statefulset Spec.Replicas: %d",
				*stOne.Definition.Spec.Replicas)

			if stOne.Definition.Spec.Replicas != nil {
				return stOne.Object.Status.ReadyReplicas == *stOne.Definition.Spec.Replicas
			}

			return stOne.Object.Status.ReadyReplicas != 0
		}

		return false
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
		"Statefulset %q in %q namespace is not ready", myStatefulsetTwo, RDSCoreConfig.WhereaboutNS)
}
