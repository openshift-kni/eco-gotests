package sniffer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/imagestream"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	scc "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/scc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	httpProtocol           = "http"
	udpProtocol            = "udp"
	snifferDeployRBACName  = "privileged-rdscore-sniffer"
	snifferDeploySAName    = "default"
	snifferRBACRole        = "system:openshift:scc:privileged"
	snifferDeploymentLabel = "rds-core=sniffer-deploy"

	tcpCaptureScript = `#!/bin/bash
tcpdump -s 0 -A -v -i %s | egrep -i 'Host:'
`
	udpCaptureScript = `#!/bin/bash
tcpdump -n -s0  -i %s port %d and udp
`
)

// CreatePacketSnifferDeployment creates packet sniffer pods on the nodes specified in scheduleOnNodes.
func CreatePacketSnifferDeployment(
	apiClient *clients.Settings,
	packetCapturePort int,
	packetCaptureProtocol,
	packetCaptureInterface,
	snifferNamespace string,
	scheduleOnNodes []string) (*deployment.Builder, error) {
	glog.V(100).Infof("Getting the image and tag for tools from the release image")

	tcpdumpImage, err := determineNetworkToolsImage(apiClient)
	if err != nil {
		return nil, err
	}
	//nolint:lll,nolintlint
	// tcpdumpImage := "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:7a45219398feb4bc69ed32ec855b07ce16fef264af74d571b230cb339a700210"
	snifferDeploymentName := fmt.Sprintf("%s-packet-sniffer", snifferNamespace)

	glog.V(100).Infof("Adding SCC privileged to the sniffer namespace")

	err = scc.AddPrivilegedSCCtoDefaultSA(snifferNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to add SCC privileged to the sniffer namespace %s: %w",
			snifferNamespace, err)
	}

	snifferDeployment, err := createHostNetworkedPacketSnifferDeployment(
		apiClient,
		packetCapturePort,
		packetCaptureProtocol,
		packetCaptureInterface,
		tcpdumpImage,
		snifferDeploymentName,
		snifferNamespace,
		scheduleOnNodes,
	)

	if err != nil {
		return snifferDeployment, err
	}

	retries := 12
	pollInterval := 5

	for i := 0; i < retries; i++ {
		snifferDeployment, err = deployment.Pull(apiClient, snifferDeploymentName, snifferNamespace)
		if err != nil {
			return nil, err
		}

		// Check if NumberReady == DesiredNumberScheduled.
		// In that case, simply return as all went well.
		if snifferDeployment.Object.Status.ReadyReplicas == snifferDeployment.Object.Status.Replicas {
			return snifferDeployment, nil
		}
		// If no port conflict error was found, simply sleep for pollInterval and then
		// check again.
		time.Sleep(time.Duration(pollInterval) * time.Second)
	}

	return snifferDeployment, fmt.Errorf("deployment still not ready after %d tries", retries)
}

// determineNetworkToolsImage will get the image and tag for network-tools from the release image.
func determineNetworkToolsImage(apiClient *clients.Settings) (string, error) {
	glog.V(100).Infof("Gathering the image and tag for tools")

	imageName := "tools"
	imageNamespace := "openshift"

	imageStreamObj, err := imagestream.Pull(apiClient, imageName, imageNamespace)

	if err != nil {
		return "", fmt.Errorf("could not find imageStream for image %s in namespace %s; %w",
			imageName, imageNamespace, err)
	}

	dockerImageString, err := imageStreamObj.GetDockerImage("latest")

	if err != nil {
		return "", fmt.Errorf("could not find dockerImage definition for the imageStream %s in namespace %s; %w",
			imageName, imageNamespace, err)
	}

	return dockerImageString, nil
}

// createHostNetworkedPacketSnifferDeployment creates a host networked pod in namespace <namespace> on node <nodeName>.
// It will start a packet sniffer, and it will log all GET request's source IP and the actual request string.
//
//nolint:funlen
func createHostNetworkedPacketSnifferDeployment(
	apiClient *clients.Settings,
	packetCapturePort int,
	packetCaptureProtocol,
	packetCaptureInterface,
	networkPacketSnifferImage,
	snifferDeploymentName,
	snifferNamespace string,
	scheduleOnHosts []string) (*deployment.Builder, error) {
	if packetCaptureProtocol != httpProtocol && packetCaptureProtocol != udpProtocol {
		return nil, fmt.Errorf("createHostNetworkedPacketSnifferDeployment supports only %s and "+
			"%s protocols, got: %s", httpProtocol, udpProtocol, packetCaptureProtocol)
	}

	glog.V(100).Infof("Creating the sniffer deployment with image %s", networkPacketSnifferImage)

	cmd := tcpCaptureScript
	if packetCaptureProtocol == udpProtocol {
		cmd = udpCaptureScript
	}

	if strings.Contains(packetCaptureInterface, ".") {
		packetCaptureInterface = strings.Split(packetCaptureInterface, ".")[0]
	}

	containerCmd := []string{
		"/bin/bash",
		"-c",
		fmt.Sprintf(cmd, packetCaptureInterface, packetCapturePort),
	}

	var err error

	glog.V(100).Infof("Checking sniffer deployment don't exist")

	err = apiobjectshelper.DeleteDeployment(apiClient, snifferDeploymentName, snifferNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to delete deployment %s from namespace %s",
			snifferDeploymentName, snifferNamespace)
	}

	glog.V(100).Infof("Sleeping 10 seconds")
	time.Sleep(10 * time.Second)

	glog.V(100).Infof("Removing ServiceAccount")

	err = apiobjectshelper.DeleteServiceAccount(apiClient, snifferDeploySAName, snifferNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to remove serviceAccount %q from namespace %q",
			snifferDeploySAName, snifferNamespace)
	}

	glog.V(100).Infof("Creating ServiceAccount")

	err = apiobjectshelper.CreateServiceAccount(apiClient, snifferDeploySAName, snifferNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to create serviceAccount %q in namespace %q",
			snifferDeploySAName, snifferNamespace)
	}

	glog.V(100).Infof("Removing Cluster RBAC")

	err = apiobjectshelper.DeleteClusterRBAC(apiClient, snifferDeployRBACName)

	if err != nil {
		return nil, fmt.Errorf("failed to delete sniffer RBAC %q", snifferDeployRBACName)
	}

	glog.V(100).Infof("Creating Cluster RBAC")

	err = apiobjectshelper.CreateClusterRBAC(apiClient, snifferDeployRBACName, snifferRBACRole,
		snifferDeploySAName, snifferNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to create sniffer RBAC %q in namespace %s",
			snifferDeployRBACName, snifferNamespace)
	}

	glog.V(100).Infof("Defining container configuration")

	deployContainer := defineTCPDumpContainer(networkPacketSnifferImage, containerCmd)

	glog.V(100).Infof("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain container definition: %w", err)
	}

	glog.V(100).Infof("Defining deployment configuration")

	deployLabelsMap := map[string]string{
		strings.Split(snifferDeploymentLabel, "=")[0]: strings.Split(snifferDeploymentLabel, "=")[1]}

	snifferDeployment := defineDeployment(
		apiClient,
		deployContainerCfg,
		snifferDeploymentName,
		snifferNamespace,
		snifferDeploySAName,
		scheduleOnHosts,
		deployLabelsMap)

	glog.V(100).Infof("Creating deployment")

	snifferDeployment, err = snifferDeployment.CreateAndWaitUntilReady(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment %s in namespace %s: %w",
			snifferDeploymentName, snifferNamespace, err)
	}

	if snifferDeployment == nil {
		return nil, fmt.Errorf("failed to create deployment %s", snifferDeploymentName)
	}

	return snifferDeployment, err
}

// sendProbesAndCheckPacketSnifferLogs sends requests from the prober pod. It then
// makes sure that the expectedHits number of requests were seen in the packetSnifferDeployment's pod logs.
// We are only interested in sending our searchString.
// We do not care about the response because:
// a) We inspect the traffic that we are sending, and we search for the unique searchString
// b) We make sure that the request leaves from one of the target IPs
// Therefore, we can send our request against any destination.
// If we test against any TCP/HTTP endpoint, we can additionally
// c) Prove that we established a bidirectional stream.
func sendProbesAndCheckPacketSnifferLogs(
	apiClient *clients.Settings,
	proberPod *pod.Builder,
	snifferNamespace,
	snifferSelectorLabel,
	targetProtocol,
	targetHost string,
	targetPort,
	probeCount,
	expectedHits int,
	logScanMaxTries int) (bool, error) {
	timeStart := time.Now()

	glog.V(100).Infof("Sending requests with a unique search string from "+
		"prober pod %s/%s", proberPod.Definition.Namespace, proberPod.Definition.Name)

	searchString, err := sendProbesToHostPort(
		proberPod,
		targetProtocol,
		targetHost,
		targetPort,
		probeCount)
	if err != nil {
		return false, err
	}

	glog.V(100).Infof("%d requests sended from prober pod %s/%s",
		probeCount, proberPod.Definition.Namespace, proberPod.Definition.Name)

	time.Sleep(time.Minute)

	for i := 0; i < logScanMaxTries; i++ {
		glog.V(100).Infof("Making sure that %d requests with search string %s were seen (try %d of %d)",
			expectedHits, searchString, i+1, logScanMaxTries)

		numberFound, err := scanPacketSnifferDeploymentPodLogs(
			apiClient,
			proberPod,
			targetProtocol,
			snifferNamespace,
			snifferSelectorLabel,
			searchString,
			timeStart,
		)

		if err != nil {
			return false, err
		}

		glog.V(100).Infof("Found number of %s is: %v", searchString, numberFound)

		if numberFound == expectedHits {
			return true, nil
		}

		if numberFound > expectedHits {
			return false, nil
		}
	}

	return false, nil
}

// scanPacketSnifferDeploymentPodLogs iterates over the pods logs and searches for searchString
// and then counts the occurrences for each found IP address.
//
//nolint:funlen
func scanPacketSnifferDeploymentPodLogs(
	apiClient *clients.Settings,
	proberPod *pod.Builder,
	targetProtocol,
	snifferNamespace,
	snifferSelectorLabel,
	searchString string,
	timeSpan time.Time) (int, error) {
	if targetProtocol != httpProtocol && targetProtocol != udpProtocol {
		return 0, fmt.Errorf("scanPacketSnifferDeploymentPodLogs supports only %s and %s protocols",
			httpProtocol, udpProtocol)
	}

	snifferPodObjects, err := pod.List(apiClient, snifferNamespace,
		metav1.ListOptions{LabelSelector: snifferSelectorLabel})
	if err != nil {
		return 0, err
	}

	matchesFound := 0

	var snifferPodObj *pod.Builder

	for _, podObj := range snifferPodObjects {
		if podObj.Object.Spec.NodeName == proberPod.Object.Spec.NodeName {
			glog.V(100).Infof("sniffer pod: %s", podObj.Definition.Name)
			snifferPodObj = podObj
		}
	}

	if snifferPodObj == nil {
		return 0, fmt.Errorf("sniffer pod not found running on the %s node", proberPod.Object.Spec.NodeName)
	}

	logStartTimestamp := time.Since(timeSpan)
	glog.V(100).Infof("\tTime duration is %s", logStartTimestamp)

	if logStartTimestamp.Abs().Seconds() < 1 {
		logStartTimestamp, err = time.ParseDuration("1s")
		if err != nil {
			return 0, fmt.Errorf("failed to parse time duration: %w", err)
		}
	}

	logs, err := snifferPodObj.GetLog(logStartTimestamp, snifferPodObj.Definition.Spec.Containers[0].Name)

	glog.V(100).Infof("debug log for %s: %s", snifferPodObj.Definition.Name, logs)

	if err != nil {
		glog.V(100).Infof("Failed to get logs from pod %q: %v",
			snifferPodObj.Definition.Name, err)

		return 0, fmt.Errorf("failed to get logs from pod %q: %w",
			snifferPodObj.Definition.Name, err)
	}

	if len(logs) != 0 {
		buf := new(bytes.Buffer)
		_, err = buf.WriteString(logs)

		if err != nil {
			return 0, fmt.Errorf("error in copying info from pod logs to buffer")
		}

		_ = buf.String()

		scanner := bufio.NewScanner(buf)
		for scanner.Scan() {
			logLine := scanner.Text()
			if strings.Contains(logLine, searchString) {
				glog.V(100).Infof("Found hit in log line: %s", logLine)
				logLineExploded := strings.Fields(logLine)

				if len(logLineExploded) != 2 {
					return 0, fmt.Errorf("unexpected logline content: %s", logLine)
				}

				matchesFound++
			}
		}

		return matchesFound, nil
	}

	return 0, fmt.Errorf("no matches found in sniffer log")
}

// sendProbesToHostPort runs curl against a http://targetHost:targetPort for the specified number of iterations.
// Returns the "targetHost:targetPort" string that will be used for the requests search.
func sendProbesToHostPort(
	proberPod *pod.Builder,
	targetProtocol,
	targetHost string,
	targetPort,
	iterations int) (string, error) {
	if proberPod == nil {
		return "", fmt.Errorf("nil pointer to proberPod was provided in sendProbesToHostPort")
	}

	if targetProtocol != httpProtocol && targetProtocol != udpProtocol {
		return "", fmt.Errorf("sendProbesToHostPort supports only %s and %s protocols",
			httpProtocol, udpProtocol)
	}

	targetURL := fmt.Sprintf("%s:%d", targetHost, targetPort)
	request := fmt.Sprintf("http://%s", targetURL)

	cmdToRun := []string{"/bin/bash", "-c", fmt.Sprintf("curl --max-time 10 -s %s", request)}
	errChan := make(chan error, iterations)

	timeout := time.Minute * 3
	err := wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second*15,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			for i := 0; i < iterations; i++ {
				output, err := proberPod.ExecCommand(cmdToRun, proberPod.Object.Spec.Containers[0].Name)

				if err != nil {
					glog.V(100).Infof("query failed. Request: %s, Output: %q, Error: %v", request, output, err)

					errChan <- fmt.Errorf("query failed. Request: %s, Output: %q, Error: %w", request, output, err)

					return false, nil
				}

				glog.V(100).Infof("Successfully executed command from within a pod %q: %v",
					proberPod.Object.Name, cmdToRun)
				glog.V(100).Infof("Command's output:\n\t%v", output.String())
			}

			return true, nil
		})

	if err != nil {
		return "", fmt.Errorf("failed to execute the command '%s' during timeout %v; %w", cmdToRun, timeout, err)
	}

	// If the above yielded any errors, then append them to a list and report them.
	if len(errChan) > 0 {
		errList := fmt.Errorf("encountered the following errors: ")
		for reportedError := range errChan {
			errList = fmt.Errorf("%w {%s}", errList, reportedError.Error())
		}

		return "", errList
	}

	return targetURL, nil
}

// SendTrafficCheckLogs is a wrapper function to reduce code duplication when probing for target IPs.
// It sends <iterations> of requests with a unique search string on the launched a new prober pod.
// It then makes sure that <expectedHits> number of hits were seen.
func SendTrafficCheckLogs(
	apiClient *clients.Settings,
	proberPodObj *pod.Builder,
	snifferNamespace,
	snifferSelectorLabel,
	targetHost,
	targetProtocol string,
	targetPort,
	iterations,
	expectedHits int) error {
	timeout := 3 * time.Minute
	err := wait.PollUntilContextTimeout(
		context.TODO(),
		3*time.Second,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			glog.V(100).Infof("Verifying that the expected number of outbound " +
				"requests can be seen in the packet sniffer logs")

			result, err := sendProbesAndCheckPacketSnifferLogs(
				apiClient,
				proberPodObj,
				snifferNamespace,
				snifferSelectorLabel,
				targetProtocol,
				targetHost,
				targetPort,
				iterations,
				expectedHits,
				10)

			if err == nil && result {
				return true, nil
			}

			return false, nil
		})

	if err != nil {
		return fmt.Errorf("expected number of outbound requests was not found in the packet sniffer logs; %w", err)
	}

	return nil
}

func defineTCPDumpContainer(cImage string, cCmd []string) *pod.ContainerBuilder {
	cName := "tcpdump"

	glog.V(100).Infof("Creating container %q", cName)

	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	glog.V(100).Infof("Defining SecurityContext")

	var trueFlag = true

	userUID := new(int64)

	*userUID = 0

	// readinessProbe := &corev1.Probe{
	// 	ProbeHandler: corev1.ProbeHandler{
	// 		Exec: &corev1.ExecAction{
	// 			Command: []string{
	// 				"echo",
	// 				"ready",
	// 			},
	// 		},
	// 	},
	// }

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

	// glog.V(100).Infof("Setting ReadinessProbe")
	//
	// deployContainer = deployContainer.WithReadinessProbe(readinessProbe)

	glog.V(100).Infof("Enable TTY and Stdin; needed for immediate log propagation")

	deployContainer = deployContainer.WithTTY(true).WithStdin(true)

	glog.V(100).Infof("%q container's  definition:\n%#v", cName, deployContainer)

	return deployContainer
}

func defineDeployment(
	apiClient *clients.Settings,
	containerConfig *corev1.Container,
	deployName, deployNs, saName string,
	scheduleOnHosts []string,
	deployLabels map[string]string) *deployment.Builder {
	glog.V(100).Infof("Defining deployment %q in %q ns", deployName, deployNs)

	nodeAffinity := corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   scheduleOnHosts,
							},
						},
					},
				},
			},
		},
	}

	snifferDeployment := deployment.NewBuilder(apiClient, deployName, deployNs, deployLabels, *containerConfig)

	glog.V(100).Infof("Assigning ServiceAccount %q to the deployment", saName)

	snifferDeployment = snifferDeployment.WithServiceAccountName(saName)

	glog.V(100).Infof("Setting Replicas count")

	replicasCnt := len(scheduleOnHosts)

	snifferDeployment = snifferDeployment.WithReplicas(int32(replicasCnt))

	snifferDeployment = snifferDeployment.WithHostNetwork(true).WithAffinity(&nodeAffinity)

	return snifferDeployment
}
