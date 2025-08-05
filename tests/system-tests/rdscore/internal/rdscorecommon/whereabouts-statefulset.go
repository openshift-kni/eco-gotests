package rdscorecommon

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"

	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	"github.com/openshift-kni/eco-goinfra/pkg/statefulset"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddrInfo represents the address information for a network interface.
type AddrInfo struct {
	Family            string `json:"family,omitempty"`
	Local             string `json:"local,omitempty"`
	Prefixlen         int    `json:"prefixlen,omitempty"`
	Broadcast         string `json:"broadcast,omitempty"`
	Scope             string `json:"scope,omitempty"`
	Label             string `json:"label,omitempty"`
	ValidLifeTime     uint32 `json:"valid_life_time,omitempty"`
	PreferredLifeTime uint32 `json:"preferred_life_time,omitempty"`
}

// NetworkInterface represents the overall structure of a single network interface.
type NetworkInterface struct {
	Ifindex     int        `json:"ifindex,omitempty"`
	LinkIndex   int        `json:"link_index,omitempty"`
	Ifname      string     `json:"ifname,omitempty"`
	Flags       []string   `json:"flags,omitempty"`
	MTU         int        `json:"mtu,omitempty"`
	Qdisc       string     `json:"qdisc,omitempty"`
	Operstate   string     `json:"operstate,omitempty"`
	Group       string     `json:"group,omitempty"`
	Txqlen      int        `json:"txqlen,omitempty"`
	LinkType    string     `json:"link_type,omitempty"`
	Address     string     `json:"address,omitempty"`
	Broadcast   string     `json:"broadcast,omitempty"`
	LinkNetnsid int        `json:"link_netnsid,omitempty"`
	AddrInfo    []AddrInfo `json:"addr_info,omitempty"`
}

const (
	// WhereaboutsReconcilerSchedule is the schedule for the whereabouts reconciler.
	WhereaboutsReconcilerSchedule = "*/3 * * * *"
	// WhereaboutsReconcilerKey is the key for the whereabouts reconciler.
	WhereaboutsReconcilerKey = "reconciler_cron_expression"
	// WhereaboutsReconcilerNamespace is the namespace for the whereabouts reconciler.
	WhereaboutsReconcilerNamespace = "openshift-multus"
	// WhereaboutsReconcilcerCMName is the name of the whereabouts reconciler configmap.
	WhereaboutsReconcilcerCMName = "whereabouts-config"
)

// getActivePods gets the active pods with the given label and namespace.
func getActivePods(podLabel, namespace string) []*pod.Builder {
	By("Checking if pods are running")

	pods := findPodWithSelector(namespace, podLabel)

	Expect(pods).ToNot(BeEmpty(), "No pods found with selector %q in %q namespace",
		podLabel, namespace)

	var activePods []*pod.Builder

	for _, _pod := range pods {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod %q in %q namespace is in phase %q",
			_pod.Object.Name, _pod.Object.Namespace, _pod.Object.Status.Phase)

		if _pod.Object.Status.Phase == corev1.PodRunning {
			activePods = append(activePods, _pod)
		}
	}

	return activePods
}

// configureWhereaboutsIPReconciler configures whereabouts IP reconciler to run every 3 minutes.
func configureWhereaboutsIPReconciler() {
	By(fmt.Sprintf("Checking if configmap %q exists in %q namespace",
		WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace))

	var ctx SpecContext

	cmWhereabouts, err := configmap.Pull(APIClient, WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace)

	if err == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Configmap %q exists in %q namespace, updating it",
			WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace)

		if oldSchedule, ok := cmWhereabouts.Object.Data[WhereaboutsReconcilerKey]; ok {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Key %q already exists in configmap %q in %q namespace, updating it",
				WhereaboutsReconcilerKey, WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace)

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Old schedule: %q", oldSchedule)

			cmWhereabouts.Object.Data[WhereaboutsReconcilerKey] = WhereaboutsReconcilerSchedule
		} else {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Key %q does not exist in configmap %q in %q namespace, adding it",
				WhereaboutsReconcilerKey, WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace)

			cmWhereabouts.Object.Data[WhereaboutsReconcilerKey] = WhereaboutsReconcilerSchedule
		}

		By(fmt.Sprintf("Updating configmap %q in %q namespace",
			WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace))

		Eventually(func() error {
			_, err := cmWhereabouts.Update()

			return err
		}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(1*time.Minute).Should(Succeed(),
			"Failed to update configmap %q in %q namespace", WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace)
	} else {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Configmap %q does not exist in %q namespace, creating it",
			WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace)

		By(fmt.Sprintf("Creating configmap %q in %q namespace",
			WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace))

		By(fmt.Sprintf("Configuring whereabouts reconciler with configmap %q in %q namespace",
			WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace))

		createConfigMap(WhereaboutsReconcilcerCMName, WhereaboutsReconcilerNamespace, map[string]string{
			WhereaboutsReconcilerKey: WhereaboutsReconcilerSchedule,
		})
	}
}

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

	configureWhereaboutsIPReconciler()

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

	By("Checking if pods are running")

	activePods := getActivePods(myStatefulsetOneLabel, RDSCoreConfig.WhereaboutNS)

	// pods := findPodWithSelector(RDSCoreConfig.WhereaboutNS, myStatefulsetOneLabel)

	// Expect(pods).ToNot(BeEmpty(), "No pods found with selector %q in %q namespace",
	// 	myStatefulsetOneLabel, RDSCoreConfig.WhereaboutNS)

	// var activePods []*pod.Builder

	// for _, _pod := range pods {
	// 	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod %q in %q namespace is in phase %q",
	// 		_pod.Object.Name, _pod.Object.Namespace, _pod.Object.Status.Phase)

	// 	if _pod.Object.Status.Phase == corev1.PodRunning {
	// 		activePods = append(activePods, _pod)
	// 	}
	// }

	Expect(len(activePods)).To(Equal(int(myStatefulsetOneReplicas)),
		"Number of active pods is not equal to number of replicas")

	By("Checking pods IP addresses")

	// NOTE: net1 is the name of the network interface in the pod, make it configurable?
	cmdGetIPAddr := []string{"/bin/sh", "-c", "ip -j addr show dev net1"}

	podWhereaboutsIPs := make(map[string][]NetworkInterface)

	for _, _pod := range activePods {
		var networkInterface []NetworkInterface

		Eventually(func() bool {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Executing command %q within pod %q in %q namespace",
				cmdGetIPAddr, _pod.Object.Name, _pod.Object.Namespace)

			addrBuffInfo, err := _pod.ExecCommand(cmdGetIPAddr, _pod.Definition.Spec.Containers[0].Name)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to execute command within pod %q in %q namespace: %s",
					_pod.Object.Name, _pod.Object.Namespace, err)

				return false
			}

			if addrBuffInfo.Len() == 0 {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Empty output from command within pod %q in %q namespace",
					_pod.Object.Name, _pod.Object.Namespace)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Unmarshalling IP addresses")

			err = json.Unmarshal(addrBuffInfo.Bytes(), &networkInterface)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to unmarshal IP addresses for pod %q in %q namespace: %s",
					_pod.Object.Name, _pod.Object.Namespace, err)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("IP addresses: %+v", networkInterface)

			podWhereaboutsIPs[_pod.Object.Name] = networkInterface

			return true
		}).WithContext(ctx).WithPolling(10*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
			"Failed to get IP addresses for pod %q in %q namespace", _pod.Object.Name, _pod.Object.Namespace)
	}

	podOneName := activePods[0].Object.Name
	podTwoName := activePods[len(activePods)-1].Object.Name

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod one %q", podOneName)
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod two %q", podTwoName)

	podsMapping := make(map[string]string)

	podsMapping[podOneName] = podTwoName
	podsMapping[podTwoName] = podOneName

	for podIndex, _pod := range activePods {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Running from %q to %q",
			_pod.Object.Name, podsMapping[_pod.Object.Name])

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod %q IP addresses: %+v",
			podsMapping[_pod.Object.Name], podWhereaboutsIPs[podsMapping[_pod.Object.Name]])

		for _, dstAddr := range podWhereaboutsIPs[podsMapping[_pod.Object.Name]][0].AddrInfo {
			if dstAddr.Family == "inet6" && dstAddr.Scope == "link" {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping link-local address %q", dstAddr.Local)

				continue
			}

			randomNumber := rand.Intn(3000)

			msgOne := fmt.Sprintf("Hello from %q to %q with random number %d",
				_pod.Object.Name, podsMapping[_pod.Object.Name], randomNumber)

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Sending data from %q to %q",
				_pod.Object.Name, dstAddr.Local)

			targetAddr := fmt.Sprintf("%s %d", dstAddr.Local, parsedPort)

			sendDataOneCmd := []string{"/bin/bash", "-c",
				fmt.Sprintf("echo '%s' | nc %s", msgOne, targetAddr)}

			var podOneResult bytes.Buffer

			timeStart := time.Now()

			Eventually(func() bool {
				podOneResult.Reset()

				podOneResult, err = _pod.ExecCommand(sendDataOneCmd, _pod.Definition.Spec.Containers[0].Name)

				return err == nil
			}).WithContext(ctx).WithPolling(10*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
				"Failed to send data from pod %q to %q", _pod.Object.Name, targetAddr)

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod %q result: %s", _pod.Object.Name, podOneResult.String())

			targetPod := activePods[len(activePods)-(podIndex+1)]

			By(fmt.Sprintf("Verifying message in pod %q", targetPod.Object.Name))

			verifyMsgInPodLogs(targetPod, msgOne, targetPod.Definition.Spec.Containers[0].Name, timeStart)
		}
	}
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

	configureWhereaboutsIPReconciler()

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

	By("Checking if pods are running")

	activePods := getActivePods(myStatefulsetTwoLabel, RDSCoreConfig.WhereaboutNS)

	Expect(len(activePods)).To(Equal(int(myStatefulsetTwoReplicas)),
		"Number of active pods is not equal to number of replicas")
}
