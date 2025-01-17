package supporttools

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	scc "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/scc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	httpProtocol     = "http"
	udpProtocol      = "udp"
	tcpCaptureScript = `#!/bin/bash
tcpdump -vvv -e -nn -i %s | egrep -i 'Host:'
`

	udpCaptureScript = `#!/bin/bash
tcpdump -n -s0  -i %s port %s and udp
`
	logScanMaxTries = 3
)

// CreateTCPDumpDeployment creates support-tools tcpdump deployment on the nodes specified in scheduleOnNodes.
func CreateTCPDumpDeployment(
	apiClient *clients.Settings,
	tdNamespace,
	tdDeploymentLabel,
	tdImage,
	packetCaptureInterface,
	packetCaptureProtocol string,
	scheduleOnNodes []string) (*deployment.Builder, error) {
	if packetCaptureProtocol != httpProtocol && packetCaptureProtocol != udpProtocol {
		return nil, fmt.Errorf("CreateTraceRouteDeployment supports only %s and "+
			"%s protocols, got: %s", httpProtocol, udpProtocol, packetCaptureProtocol)
	}

	glog.V(100).Infof("Getting the image and tag for tools from the release image")

	stDeploymentName := "tcpdump-tool"

	glog.V(100).Infof("Create support-tools namespace %s", tdNamespace)

	err := ensureNamespaceExists(apiClient, tdNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create support-tools namespace %s: %w", tdNamespace, err)
	}

	glog.V(100).Infof("Adding SCC privileged to the support-tools namespace")

	err = scc.AddPrivilegedSCCtoDefaultSA(tdNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to add SCC privileged to the supporttools namespace %s: %w",
			tdNamespace, err)
	}

	stDeployment, err := createTCPDumpDeployment(
		apiClient,
		tdImage,
		stDeploymentName,
		tdNamespace,
		tdDeploymentLabel,
		packetCaptureInterface,
		packetCaptureProtocol,
		scheduleOnNodes,
	)

	if err != nil {
		return stDeployment, err
	}

	retries := 3
	pollInterval := 5

	for i := 0; i < retries; i++ {
		stDeployment, err = deployment.Pull(apiClient, stDeploymentName, tdNamespace)
		if err != nil {
			return nil, err
		}

		if stDeployment.Object.Status.ReadyReplicas == stDeployment.Object.Status.Replicas {
			return stDeployment, nil
		}

		time.Sleep(time.Duration(pollInterval) * time.Second)
	}

	return stDeployment, fmt.Errorf("deployment still not ready after %d tries", retries)
}

// createTCPDumpDeployment creates a support-tools tcpdump pod in namespace <namespace> on node <nodeName>.
//
//nolint:funlen
func createTCPDumpDeployment(
	apiClient *clients.Settings,
	stImage,
	stDeploymentName,
	stNamespace,
	stDeploymentLabel,
	packetCaptureInterface,
	packetCaptureProtocol string,
	scheduleOnHosts []string) (*deployment.Builder, error) {
	glog.V(100).Infof("Creating the support-tools tcpdump deployment with image %s", stImage)

	var err error

	glog.V(100).Infof("Checking support-tools deployment don't exist")

	err = apiobjectshelper.DeleteDeployment(apiClient, stDeploymentName, stNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to delete deployment %s from namespace %s",
			stDeploymentName, stNamespace)
	}

	glog.V(100).Infof("Sleeping 10 seconds")
	time.Sleep(10 * time.Second)

	glog.V(100).Infof("Removing ServiceAccount")

	err = apiobjectshelper.DeleteServiceAccount(apiClient, supportToolsDeploySAName, stNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to remove serviceAccount %q from namespace %q",
			supportToolsDeploySAName, stNamespace)
	}

	glog.V(100).Infof("Creating ServiceAccount")

	err = apiobjectshelper.CreateServiceAccount(apiClient, supportToolsDeploySAName, stNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to create serviceAccount %q in namespace %q",
			supportToolsDeploySAName, stNamespace)
	}

	glog.V(100).Infof("Removing Cluster RBAC")

	err = apiobjectshelper.DeleteClusterRBAC(apiClient, supportToolsDeployRBACName)

	if err != nil {
		return nil, fmt.Errorf("failed to delete supporttools RBAC %q", supportToolsDeployRBACName)
	}

	glog.V(100).Infof("Creating Cluster RBAC")

	err = apiobjectshelper.CreateClusterRBAC(apiClient, supportToolsDeployRBACName, supportToolsRBACRole,
		supportToolsDeploySAName, stNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to create supporttools RBAC %q in namespace %s",
			supportToolsDeployRBACName, stNamespace)
	}

	glog.V(100).Infof("Defining container configuration")

	deployContainer := defineTCPDumpContainer(
		stImage,
		packetCaptureInterface,
		packetCaptureProtocol)

	glog.V(100).Infof("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain container definition: %w", err)
	}

	glog.V(100).Infof("Defining deployment configuration")

	deployLabelsMap := map[string]string{
		strings.Split(stDeploymentLabel, "=")[0]: strings.Split(stDeploymentLabel, "=")[1]}

	stDeployment := defineDeployment(
		apiClient,
		deployContainerCfg,
		stDeploymentName,
		stNamespace,
		supportToolsDeploySAName,
		scheduleOnHosts,
		deployLabelsMap)

	glog.V(100).Infof("Creating deployment")

	stDeployment, err = stDeployment.CreateAndWaitUntilReady(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment %s in namespace %s: %w",
			stDeploymentName, stNamespace, err)
	}

	if stDeployment == nil {
		return nil, fmt.Errorf("failed to create deployment %s", stDeploymentName)
	}

	return stDeployment, err
}

func defineTCPDumpContainer(
	cImage,
	packetCaptureInterface,
	packetCaptureProtocol string) *pod.ContainerBuilder {
	cName := "tcpdump"

	cmd := tcpCaptureScript
	if packetCaptureProtocol == udpProtocol {
		cmd = udpCaptureScript
	}

	cCmd := []string{
		"/bin/bash",
		"-c",
		fmt.Sprintf(cmd, packetCaptureInterface),
	}

	glog.V(100).Infof("Creating container %q", cName)

	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	glog.V(100).Infof("Defining SecurityContext")

	var trueFlag = true

	userUID := new(int64)

	*userUID = 0

	securityContext := &corev1.SecurityContext{
		RunAsUser:  userUID,
		Privileged: &trueFlag,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{
				"SETFCAP",
				"CAP_NET_RAW",
				"CAP_NET_ADMIN",
			},
		},
	}

	glog.V(100).Infof("Setting SecurityContext")

	deployContainer = deployContainer.WithSecurityContext(securityContext)

	glog.V(100).Infof("Dropping ALL security capability")

	deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)

	glog.V(100).Infof("Enable TTY and Stdin; needed for immediate log propagation")

	deployContainer = deployContainer.WithTTY(true).WithStdin(true)

	glog.V(100).Infof("%q container's  definition:\n%#v", cName, deployContainer)

	return deployContainer
}

// SendTrafficCheckTCPDumpLogs is a wrapper function to reduce code duplication when probing for target IPs.
func SendTrafficCheckTCPDumpLogs(
	apiClient *clients.Settings,
	proberPodObj *pod.Builder,
	tcpdumpPodNamespace,
	tcpdumpPodSelectorLabel,
	targetHost,
	targetProtocol,
	targetPort string) error {
	timeout := 3 * time.Minute
	err := wait.PollUntilContextTimeout(
		context.TODO(),
		3*time.Second,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			glog.V(100).Infof("Verifying that the expected outbound request can be seen in the tcpdump logs")

			result, err := sendProbeAndCheckTCPDumpLogs(
				apiClient,
				proberPodObj,
				tcpdumpPodNamespace,
				tcpdumpPodSelectorLabel,
				targetProtocol,
				targetHost,
				targetPort)

			if err == nil && result {
				return true, nil
			}

			return false, nil
		})

	if err != nil {
		return fmt.Errorf("expected request was not found in the packet sniffer logs; %w", err)
	}

	return nil
}

// sendProbeAndCheckTCPDumpLogs sends requests from the prober pod.
func sendProbeAndCheckTCPDumpLogs(
	apiClient *clients.Settings,
	proberPod *pod.Builder,
	tcpdumpPodNamespace,
	tcpdumpPodSelectorLabel,
	targetProtocol,
	targetHost,
	targetPort string) (bool, error) {
	timeStart := time.Now()

	glog.V(100).Infof("Sending request with a unique search string from "+
		"prober pod %s/%s", proberPod.Definition.Namespace, proberPod.Definition.Name)

	searchString, err := sendProbeToHostPort(
		proberPod,
		targetProtocol,
		targetHost,
		targetPort)
	if err != nil {
		return false, err
	}

	glog.V(100).Infof("request sended from the prober pod %s/%s",
		proberPod.Definition.Namespace, proberPod.Definition.Name)

	time.Sleep(time.Minute)

	for i := 0; i < logScanMaxTries; i++ {
		glog.V(100).Infof("Making sure that request with the search string %s were seen (try %d of %d)",
			searchString, i+1, logScanMaxTries)

		numberFound, _, err := ScanTCPDumpPodLogs(
			apiClient,
			proberPod,
			targetProtocol,
			tcpdumpPodNamespace,
			tcpdumpPodSelectorLabel,
			searchString,
			timeStart)

		if err != nil {
			return false, err
		}

		if numberFound >= 1 {
			glog.V(100).Infof("The searching string %s was found in log %d times", searchString, numberFound)

			return true, nil
		}
	}

	return false, nil
}

// ScanTCPDumpPodLogs iterates over the pods logs and searches for searchString and then counts the occurrences.
//
//nolint:funlen
func ScanTCPDumpPodLogs(
	apiClient *clients.Settings,
	proberPod *pod.Builder,
	targetProtocol,
	tcpdumpPodNamespace,
	tcpdumpPodSelectorLabel,
	searchString string,
	timeSpan time.Time) (int, string, error) {
	if targetProtocol != httpProtocol && targetProtocol != udpProtocol {
		glog.V(100).Infof("the ScanTCPDumpPodLogs supports only %s and %s protocols",
			httpProtocol, udpProtocol)

		return 0, "", fmt.Errorf("the ScanTCPDumpPodLogs supports only %s and %s protocols",
			httpProtocol, udpProtocol)
	}

	tcpdumpPodObjects, err := pod.List(apiClient, tcpdumpPodNamespace,
		metav1.ListOptions{LabelSelector: tcpdumpPodSelectorLabel})
	if err != nil {
		glog.V(100).Infof("failed to list pod objects: %v", err)

		return 0, "", fmt.Errorf("failed to list pod objects: %w", err)
	}

	matchesFound := 0

	var tcpdumpPodObj *pod.Builder

	for _, podObj := range tcpdumpPodObjects {
		if podObj.Object.Spec.NodeName == proberPod.Object.Spec.NodeName {
			glog.V(100).Infof("tcpdump pod: %s", podObj.Definition.Name)
			tcpdumpPodObj = podObj
		}
	}

	if tcpdumpPodObj == nil {
		glog.V(100).Infof("failed to find a tcpdump pod on the %s node", proberPod.Object.Spec.NodeName)

		return 0, "", fmt.Errorf("tcpdump pod not found on the %s node", proberPod.Object.Spec.NodeName)
	}

	var tcpdumpLog string

	err = wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second,
		time.Minute,
		true,
		func(ctx context.Context) (bool, error) {
			logStartTimestamp := time.Since(timeSpan)
			glog.V(100).Infof("\tTime duration is %s", logStartTimestamp)

			if logStartTimestamp.Abs().Seconds() < 1 {
				logStartTimestamp, err = time.ParseDuration("1s")
				if err != nil {
					glog.V(100).Infof("failed to parse the time duration: %v", err)

					return false, nil
				}
			}

			tcpdumpLog, err = tcpdumpPodObj.GetLog(logStartTimestamp, tcpdumpPodObj.Definition.Spec.Containers[0].Name)

			if err != nil {
				glog.V(100).Infof("Failed to get tcpdumpLog from pod %q, log %s: %v",
					tcpdumpPodObj.Definition.Name, tcpdumpLog, err)

				return false, nil
			}

			if len(tcpdumpLog) == 0 {
				glog.V(100).Infof("tcpdumpLog from pod %q is empty, log %s: %v",
					tcpdumpPodObj.Definition.Name, tcpdumpLog, err)

				return false, nil
			}

			return true, nil
		})

	glog.V(100).Infof("debug log for %s: -%s-", tcpdumpPodObj.Definition.Name, tcpdumpLog)
	glog.V(100).Infof("len(tcpdumpLog) = %d", len(tcpdumpLog))

	if len(tcpdumpLog) != 0 {
		buf := new(bytes.Buffer)
		_, err = buf.WriteString(tcpdumpLog)

		if err != nil {
			glog.V(100).Infof("error in copying info from pod tcpdumpLog to buffer: %v", err)

			return 0, tcpdumpLog, fmt.Errorf("error in copying info from pod tcpdumpLog to buffer: %w", err)
		}

		_ = buf.String()

		scanner := bufio.NewScanner(buf)
		for scanner.Scan() {
			logLine := scanner.Text()
			if strings.Contains(logLine, searchString) {
				glog.V(100).Infof("Found hit in log line: %s", logLine)
				logLineExploded := strings.Fields(logLine)

				if len(logLineExploded) != 2 {
					return 0, tcpdumpLog, fmt.Errorf("unexpected logline content: %s", logLine)
				}

				matchesFound++
			}
		}

		return matchesFound, tcpdumpLog, nil
	}

	return 0, tcpdumpLog, fmt.Errorf("no matches found in tcpdump log")
}

// sendProbeToHostPort runs curl against a http://targetHost:targetPort.
// Returns the "targetHost:targetPort" string that will be used for the requests search.
func sendProbeToHostPort(
	proberPod *pod.Builder,
	targetProtocol,
	targetHost,
	targetPort string) (string, error) {
	if proberPod == nil {
		return "", fmt.Errorf("nil pointer to proberPod was provided in sendProbeToHostPort")
	}

	if targetProtocol != httpProtocol && targetProtocol != udpProtocol {
		return "", fmt.Errorf("sendProbeToHostPort supports only %s and %s protocols",
			httpProtocol, udpProtocol)
	}

	targetURL := fmt.Sprintf("%s:%s", targetHost, targetPort)
	request := fmt.Sprintf("http://%s", targetURL)

	cmdToRun := []string{"/bin/bash", "-c", fmt.Sprintf("curl --max-time 10 -s %s", request)}

	var errorMsg error

	timeout := time.Minute * 3
	err := wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second*15,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			output, err := proberPod.ExecCommand(cmdToRun, proberPod.Object.Spec.Containers[0].Name)

			if err != nil {
				glog.V(100).Infof("query failed. Request: %s, Output: %q, Error: %v", request, output, err)

				errorMsg = fmt.Errorf("query failed. Request: %s, Output: %q, Error: %w", request, output, err)

				return false, nil
			}

			glog.V(100).Infof("Successfully executed command from within a pod %q: %v",
				proberPod.Object.Name, cmdToRun)
			glog.V(100).Infof("Command's output:\n\t%v", output.String())

			return true, nil
		})

	if err != nil {
		return "", fmt.Errorf("failed to execute the command '%s' during timeout %v; %w", cmdToRun, timeout, err)
	}

	if errorMsg != nil {
		return "", errorMsg
	}

	return targetURL, nil
}
