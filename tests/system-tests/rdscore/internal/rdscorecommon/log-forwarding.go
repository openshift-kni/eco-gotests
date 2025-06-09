package rdscorecommon

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"k8s.io/apimachinery/pkg/util/wait"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clusterlogging"
	"github.com/openshift-kni/eco-goinfra/pkg/dns"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

const (
	kcatDeploymentName      = "kcat"
	kcatDeploymentNamespace = "default"
	logMessageCnt           = 1000
)

var (
	logTypes = []string{"audit", "infrastructure"}
)

type kafkaRecord struct {
	GeneralTimestamp string `json:"@timestamp"`
	Annotations      struct {
		Decision string `json:"authorization.k8s.io/decision,omitempty"`
		Reason   string `json:"authorization.k8s.io/reason,omitempty"`
	} `json:"annotations,omitempty"`
	Hostname   string `json:"hostname"`
	Kubernetes struct {
		Annotations struct {
			PodNetwork    string `json:"k8s.ovn.org/pod-networks,omitempty"`
			NetworkStatus string `json:"k8s.v1.cni.cncf.io/network-status,omitempty"`
			SCC           string `json:"openshift.io/scc,omitempty"`
		} `json:"annotations,omitempty"`
		ContainerID       string `json:"container_id,omitempty"`
		ContainerImage    string `json:"container_image,omitempty"`
		ContainerImageID  string `json:"container_image_id,omitempty"`
		ContainerIOStream string `json:"container_iostream,omitempty"`
		ContainerName     string `json:"container_name,omitempty"`
		Labels            struct {
			App             string `json:"app,omitempty"`
			PodTemplateHash string `json:"pod-template-hash,omitempty"`
		} `json:"labels,omitempty"`
		NamespaceID     string `json:"namespace_id,omitempty"`
		NamespaceLabels struct {
		} `json:"namespace_labels,omitempty"`
		NamespaceName string `json:"namespace_name,omitempty"`
		PodID         string `json:"pod_id,omitempty"`
		PodIP         string `json:"pod_ip,omitempty"`
		PodName       string `json:"pod_name,omitempty"`
		PodOwner      string `json:"pod_owner,omitempty"`
	} `json:"kubernetes,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
	AuditID    string `json:"auditID,omitempty"`
	AuditLevel string `json:"k8s_audit_level,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Level      string `json:"level"`
	LogSource  string `json:"log_source"`
	LogType    string `json:"log_type"`
	Message    string `json:"message,omitempty"`
	ObjectRef  struct {
		APIGroup        string `json:"apiGroup,omitempty"`
		APIVersion      string `json:"apiVersion,omitempty"`
		Name            string `json:"name,omitempty"`
		Namespace       string `json:"namespace,omitempty"`
		Resource        string `json:"resource,omitempty"`
		ResourceVersion string `json:"resourceVersion,omitempty"`
		UID             string `json:"uid,omitempty"`
	} `json:"objectRef,omitempty"`
	Openshift struct {
		ClusterID string `json:"cluster_id,omitempty"`
		Labels    struct {
			RDS      string `json:"rds,omitempty"`
			SiteName string `json:"sitename"`
			SiteUUID string `json:"siteuuid"`
		} `json:"labels,omitempty"`
		Sequence int64 `json:"sequence,omitempty"`
	} `json:"openshift,omitempty"`
	RequestReceivedTimestamp string `json:"requestReceivedTimestamp,omitempty"`
	RequestURI               string `json:"requestURI,omitempty"`
	ResponseStatus           struct {
		Code     int `json:"code,omitempty"`
		Metadata struct {
		} `json:"metadata,omitempty"`
	} `json:"responseStatus,omitempty"`
	SourceIps      []string `json:"sourceIPs,omitempty"`
	Stage          string   `json:"stage,omitempty"`
	StageTimestamp string   `json:"stageTimestamp,omitempty"`
	Timestamp      string   `json:"timestamp,omitempty"`
	User           struct {
		Extra struct {
			CredentialID []string `json:"authentication.kubernetes.io/credential-id,omitempty"`
			NodeName     []string `json:"authentication.kubernetes.io/node-name,omitempty"`
			NodeUID      []string `json:"authentication.kubernetes.io/node-uid,omitempty"`
			PodName      []string `json:"authentication.kubernetes.io/pod-name,omitempty"`
			PodUID       []string `json:"authentication.kubernetes.io/pod-uid,omitempty"`
		} `json:"extra,omitempty"`
		Groups   []string `json:"groups,omitempty"`
		UID      string   `json:"uid,omitempty"`
		UserName string   `json:"userName,omitempty"`
	} `json:"user,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	Verb      string `json:"verb,omitempty"`
}

func getPodObjectByNamePattern(apiClient *clients.Settings, podNamePattern, podNamespace string) (*pod.Builder, error) {
	var podObj *pod.Builder

	err := wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second*5,
		time.Minute*1,
		true,
		func(ctx context.Context) (bool, error) {
			podObjList, err := pod.ListByNamePattern(apiClient, podNamePattern, podNamespace)
			if err != nil {
				glog.V(100).Infof("Error getting pod object by name pattern %q in namespace %q: %v",
					podNamePattern, podNamespace, err)

				return false, nil
			}

			if len(podObjList) == 0 {
				glog.V(100).Infof("No pods %s were found in namespace %q", podNamePattern, podNamespace)

				return false, nil
			}

			if len(podObjList) > 1 {
				glog.V(100).Infof("Wrong pod %s count %d was found in namespace %q", podNamePattern, len(podObjList), podNamespace)

				for _, _pod := range podObjList {
					glog.V(100).Infof("Pod %q is in %q phase", _pod.Definition.Name, _pod.Object.Status.Phase)
				}

				return false, nil
			}

			podObj = podObjList[0]

			return true, nil
		})

	if err != nil {
		glog.V(100).Infof("Failed to retrieve pod %q in namespace %q: %v",
			podNamePattern, podNamespace, err)

		return nil, fmt.Errorf("failed to retrieve pod %q in namespace %q: %w",
			podNamePattern, podNamespace, err)
	}

	err = podObj.WaitUntilReady(time.Second * 30)
	if err != nil {
		glog.V(100).Infof("The pod-level bonded pod %s in namespace %s is not in Ready state: %v",
			podNamePattern, podNamespace, err)

		return nil, fmt.Errorf("the pod-level bonded pod %s in namespace %s is not in Ready state: %w",
			podNamePattern, podNamespace, err)
	}

	return podObj, nil
}

// VerifyLogForwardingToKafka Verify cluster log forwarding to the Kafka aggregator.
//
//nolint:funlen
func VerifyLogForwardingToKafka() {
	By("Insure CLO deployed")

	var ctx SpecContext

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verify CLO namespace %s defined", rdscoreparams.CLONamespace)

	err := apiobjectshelper.VerifyNamespaceExists(APIClient, rdscoreparams.CLONamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull namespace %q; %v",
		rdscoreparams.CLONamespace, err))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Verify CLO deployment %s defined in namespace %s",
		rdscoreparams.CLODeploymentName, rdscoreparams.CLONamespace)

	err = apiobjectshelper.VerifyOperatorDeployment(APIClient,
		rdscoreparams.CLOName,
		rdscoreparams.CLODeploymentName,
		rdscoreparams.CLONamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("operator deployment %s failure in the namespace %s; %v",
			rdscoreparams.CLOName, rdscoreparams.CLONamespace, err))

	By("Retrieve kafka server URL")
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Retrieve Kafka server URL from the ClusterLogForwarder")

	clusterLogForwarder, err := clusterlogging.PullClusterLogForwarder(
		APIClient, rdscoreparams.CLOInstanceName, rdscoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(
		"Failed to retrieve ClusterLogForwarder %s from the namespace %q; %v",
		rdscoreparams.CLOInstanceName, rdscoreparams.CLONamespace, err))

	clfOutput := clusterLogForwarder.Object.Spec.Outputs
	Expect(len(clfOutput)).ToNot(Equal(0), fmt.Sprintf(
		"No collector defined in the ClusterLogForwarder %s from the namespace %q",
		rdscoreparams.CLOInstanceName, rdscoreparams.CLONamespace))

	var kafkaURL, kafkaUser string

	for _, collector := range clfOutput {
		if collector.Type == "kafka" {
			clfKafkaURL := collector.Kafka.URL

			glog.V(100).Infof("collector.URL: %s", clfKafkaURL)

			kafkaURL = strings.Split(clfKafkaURL, "/")[2]
			kafkaUser = strings.Split(clfKafkaURL, "/")[3]

			break
		}
	}

	kafkaLogsLabel := RDSCoreConfig.KafkaLogsLabel

	if kafkaLogsLabel == "" {
		By("Getting cluster domain")

		clusterDNS, err := dns.Pull(APIClient)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(
			"Failed to retrieve clusterDNS object cluster from the namespace default; %v", err))

		clusterDomain := clusterDNS.Object.Spec.BaseDomain
		Expect(clusterDomain).ToNot(Equal(""), "cluster domain is empty")

		glog.V(100).Infof("DEBUG: clusterDomain: %s", clusterDomain)

		kafkaLogsLabel = clusterDomain
	}

	By("Build query request command")

	cmdToRun := []string{"/bin/sh", "-c", fmt.Sprintf("kcat -b %s -C -t %s -C -q -o end -c %d | grep %s",
		kafkaURL, kafkaUser, logMessageCnt, kafkaLogsLabel)}

	By("Retrieve kcat pod object")

	kcatPodObj, err := getPodObjectByNamePattern(APIClient, kcatDeploymentName, kcatDeploymentNamespace)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to retrieve %s pod object from namespace %s: %v",
			kcatDeploymentName, kcatDeploymentNamespace, err))

	By("Retrieve logs forwarded to the kafka")

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Execute command: %q", cmdToRun)

	var logMessages []kafkaRecord

	Eventually(func() bool {
		output, err := kcatPodObj.ExecCommand(cmdToRun, kcatPodObj.Object.Spec.Containers[0].Name)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				"Error running command from within a pod %q in namespace %q: %v",
				kcatPodObj.Definition.Name, kcatPodObj.Definition.Namespace, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
			"Successfully executed command from within a pod %q in namespace %q",
			kcatPodObj.Definition.Name, kcatPodObj.Definition.Namespace)

		result := output.String()

		if result == "" {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				"Empty result received from within a pod %q in namespace %q",
				kcatPodObj.Definition.Name, kcatPodObj.Definition.Namespace)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Analyse received logs:\n\t%v", result)

		result = strings.TrimSpace(result)

		for _, line := range strings.Split(result, "\n") {
			var logMessage kafkaRecord

			err = json.Unmarshal([]byte(line), &logMessage)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error unmarshalling kafka record %q: %v", line, err)

				return false
			}

			logMessages = append(logMessages, logMessage)
		}

		if len(logMessages) == 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("No log messages forwarded to the kafka %s found", kafkaURL)

			return false
		}

		for _, logType := range logTypes {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
				"Verify %s type log messages were forwarded to the kafka server %s", logType, kafkaURL)

			messageCnt := 0

			for _, logMessage := range logMessages {
				if logMessage.LogType == logType {
					messageCnt++
				}
			}

			if messageCnt == 0 {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof(
					"No log messages of %s type forwarded to the kafka %s were found", kafkaURL, logType)

				return false
			}

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found %d %s log messages forwarded to the kafka server",
				messageCnt, logType)
		}

		return true
	}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(6*time.Minute).Should(BeTrue(),
		"pods matching label() still present")
}
