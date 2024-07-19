package vcorecommon

import (
	"fmt"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterlogging"
	"github.com/openshift-kni/eco-goinfra/pkg/storage"
	"time"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"
	corev1 "k8s.io/api/core/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/console"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
	clov1 "github.com/openshift/cluster-logging-operator/api/logging/v1"
)

var (
	loggingCollectorPodNamePattern = "collector"
	loggingLokiOnePodNamePattern   = []string{
		"logging-loki-compactor",
		"logging-loki-index-gateway",
		"logging-loki-ingester",
		"logging-view-plugin",
	}
	loggingLokiTwoPodsNamePattern = []string{
		"logging-loki-distributor",
		"logging-loki-gateway",
		"logging-loki-querier",
		"logging-loki-query-frontend",
	}
)

// VerifyLokiSuite container that contains tests for LokiStack and ClusterLogging verification.
func VerifyLokiSuite() {
	Describe(
		"LokiStack and Cluster Logging validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.CLONamespace),
				Label("loki"), VerifyCLONamespaceExists)

			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.LokiNamespace),
				Label("loki"), VerifyLokiNamespaceExists)

			It("Verify Loki Operator successfully installed",
				Label("loki"), reportxml.ID("74913"), VerifyLokiDeployment)

			It("Verify ClusterLogging Operator successfully installed",
				Label("loki"), reportxml.ID("73678"), VerifyCLODeployment)

			It("Create ObjectBucketClaim config",
				Label("loki"), reportxml.ID("74914"), CreateObjectBucketClaim)

			It("Create LokiStack instance",
				Label("loki"), reportxml.ID("74915"), CreateLokiStackInstance)

			It(fmt.Sprintf("Verify Cluster Logging instance %s is running in namespace %s",
				vcoreparams.CLOInstanceName, vcoreparams.CLONamespace),
				Label("loki"), reportxml.ID("59494"), CreateCLOInstance)
		})
}

// VerifyCLONamespaceExists asserts namespace for ClusterLogging Operator exists.
func VerifyCLONamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.CLONamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.CLONamespace))
} // func VerifyCLONamespaceExists (ctx SpecContext)

// VerifyLokiNamespaceExists asserts namespace for ElasticSearch Operator exists.
func VerifyLokiNamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.LokiNamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to pull %q namespace", vcoreparams.LokiNamespace))
} // func VerifyLokiNamespaceExists (ctx SpecContext)

// VerifyLokiDeployment asserts ElasticSearch Operator successfully installed.
func VerifyLokiDeployment(ctx SpecContext) {
	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		vcoreparams.LokiOperatorSubscriptionName,
		vcoreparams.LokiOperatorDeploymentName,
		vcoreparams.LokiNamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Loki operator deployment %s failure in the namespace %s; %v",
			vcoreparams.LokiOperatorDeploymentName, vcoreparams.LokiNamespace, err))
} // func VerifyLokiDeployment (ctx SpecContext)

// VerifyCLODeployment asserts ClusterLogging Operator successfully installed.
func VerifyCLODeployment(ctx SpecContext) {
	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		vcoreparams.CLOName,
		vcoreparams.CLODeploymentName,
		vcoreparams.CLONamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("operator deployment %s failure in the namespace %s; %v",
			vcoreparams.CLOName, vcoreparams.CLONamespace, err))
} // func VerifyCLODeployment (ctx SpecContext)

// CreateObjectBucketClaim asserts the ObjectBucketClaim created.
func CreateObjectBucketClaim(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Create an objectBucketClaim for openshift-storage.noobaa.io")

	var err error

	objectBucketClaimObj := storage.NewObjectBucketClaimBuilder(APIClient,
		vcoreparams.ObjectBucketClaimName,
		vcoreparams.CLONamespace)

	if objectBucketClaimObj.Exists() {
		err := objectBucketClaimObj.Delete()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete objectBucketClaim %s from namespace %s; %v",
				vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace, err))
	}

	_, err = objectBucketClaimObj.
		WithGenerateBucketName(vcoreparams.ObjectBucketClaimName).
		WithStorageClassName("openshift-storage.noobaa.io").Create()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("failed to create objectBucketClaim %s in namespace %s due to %v",
			vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace, err))
} // func CreateObjectBucketClaim (ctx SpecContext)

// CreateLokiStackInstance asserts the LokiStack instance created.
//
//nolint:goconst,funlen
func CreateLokiStackInstance(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Create a LokiStack instance")

	var err error

	lokiSecretObj := secret.NewBuilder(APIClient,
		vcoreparams.LokiSecretName,
		vcoreparams.CLONamespace,
		corev1.SecretTypeOpaque)

	if lokiSecretObj.Exists() {
		err := lokiSecretObj.Delete()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete loki secret %s from namespace %s; %v",
				vcoreparams.LokiSecretName, vcoreparams.CLONamespace, err))
	}

	glog.V(vcoreparams.VCoreLogLevel).Infof("Create loki secret %s in namespace %s",
		vcoreparams.LokiSecretName, vcoreparams.CLONamespace)

	err = await.WaitUntilConfigMapCreated(APIClient,
		vcoreparams.ObjectBucketClaimName,
		vcoreparams.CLONamespace, time.Minute*15)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("configMap %s was not created in namespace %s; "+
			"check objectBucketClaim %s in namespace %s configuration; %v",
			vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace,
			vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace, err))

	objectBucketClaimConfigMap, err := configmap.Pull(APIClient,
		vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("configMap %s not found in namespace %s; "+
			"check objectBucketClaim %s in namespace %s configuration; %v",
			vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace,
			vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace, err))

	cmData := objectBucketClaimConfigMap.Object.Data
	bucketHost := cmData["BUCKET_HOST"]
	bucketName := cmData["BUCKET_NAME"]
	bucketPort := cmData["BUCKET_PORT"]

	objectBucketClaimSecret, err := secret.Pull(APIClient,
		vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("secret %s not found in namespace %s; "+
			"check objectBucketClaim %s in namespace %s configuration; %v",
			vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace,
			vcoreparams.ObjectBucketClaimName, vcoreparams.CLONamespace, err))

	secretData := objectBucketClaimSecret.Object.Data
	accessKeyID := string(secretData["AWS_ACCESS_KEY_ID"])
	secretAccessKey := string(secretData["AWS_SECRET_ACCESS_KEY"])

	stringData := map[string]string{
		"bucketnames":       bucketName,
		"endpoint":          fmt.Sprintf("https://%s:%s", bucketHost, bucketPort),
		"region":            "",
		"access_key_id":     accessKeyID,
		"access_key_secret": secretAccessKey,
	}

	_, err = lokiSecretObj.WithStringData(stringData).Create()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to create secret %s in namespace %s due to: %v",
			vcoreparams.LokiSecretName, vcoreparams.CLONamespace, err))

	lokiStackObj := clusterlogging.NewLokiStackBuilder(APIClient,
		vcoreparams.LokiStackName,
		vcoreparams.CLONamespace)

	if lokiStackObj.Exists() {
		_, err = lokiStackObj.Delete()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete lokiStack %s from namespace %s; %v",
				vcoreparams.LokiStackName, vcoreparams.CLONamespace, err))
	}

	storage := lokiv1.ObjectStorageSpec{
		Schemas: []lokiv1.ObjectStorageSchema{{
			EffectiveDate: lokiv1.StorageSchemaEffectiveDateFormat,
			Version:       lokiv1.ObjectStorageSchemaV13,
		}},
		Secret: lokiv1.ObjectStorageSecretSpec{
			Type: "s3",
			Name: vcoreparams.LokiSecretName,
		},
		TLS: &lokiv1.ObjectStorageTLSSpec{CASpec: lokiv1.CASpec{
			CAKey: "caName",
			CA:    "openshift-service-ca.crt",
		},
		},
	}

	lokiComponent := lokiv1.LokiComponentSpec{
		NodeSelector: map[string]string{"node-role.kubernetes.io/infra": ""},
		Tolerations: []corev1.Toleration{{
			Key:      "node-role.kubernetes.io/infra",
			Operator: "Exists",
		}},
	}

	template := lokiv1.LokiTemplateSpec{
		Compactor:     &lokiComponent,
		Distributor:   &lokiComponent,
		Ingester:      &lokiComponent,
		Querier:       &lokiComponent,
		QueryFrontend: &lokiComponent,
		Gateway:       &lokiComponent,
		IndexGateway:  &lokiComponent,
		Ruler:         &lokiComponent,
	}

	_, err = lokiStackObj.
		WithSize(lokiv1.SizeOneXSmall).
		WithManagementState(lokiv1.ManagementStateManaged).
		WithStorage(storage).
		WithStorageClassName(vcoreparams.StorageClassName).
		WithTenants(lokiv1.TenantsSpec{Mode: lokiv1.OpenshiftLogging}).
		WithTemplate(template).
		WithLimits(lokiv1.LimitsSpec{
			Global: &lokiv1.LimitsTemplateSpec{
				Retention: &lokiv1.RetentionLimitSpec{
					Days: 7,
				},
			},
		}).Create()
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("failed to create lokiStack instance %s in namespace %s due to %v",
			vcoreparams.LokiStackName, vcoreparams.CLONamespace, err))
} // func CreateLokiStackInstance (ctx SpecContext)

// CreateCLOInstance asserts ClusterLogging instance can be created and running.
//
//nolint:funlen
func CreateCLOInstance(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify Cluster Logging instance %s is running in namespace %s",
		vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	var err error

	clusterLoggingObj := clusterlogging.NewBuilder(APIClient, vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	if clusterLoggingObj.Exists() {
		err := clusterLoggingObj.Delete()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to delete ClusterLogging %s csv name from the %s namespace",
			vcoreparams.CLOInstanceName, vcoreparams.CLONamespace))
	}

	glog.V(vcoreparams.VCoreLogLevel).Infof("Create new Cluster Logging instance %s in namespace %s",
		vcoreparams.CLOInstanceName, vcoreparams.CLONamespace)

	_, err = clusterLoggingObj.
		WithManagementState(clov1.ManagementStateManaged).
		WithLogStore(clov1.LogStoreSpec{
			Type:      "lokistack",
			LokiStack: clov1.LokiStackStoreSpec{Name: "logging-loki"},
		}).
		WithCollection(clov1.CollectionSpec{
			Type: "vector",
			CollectorSpec: clov1.CollectorSpec{
				Tolerations: []corev1.Toleration{{
					Key:      "node-role.kubernetes.io/infra",
					Operator: "Exists",
				}, {
					Key:      "node.ocs.openshift.io/storage",
					Operator: "Equal",
					Value:    "true",
					Effect:   "NoSchedule",
				}},
			}}).
		WithVisualization(clov1.VisualizationSpec{
			Type:         "ocp-console",
			NodeSelector: map[string]string{"node-role.kubernetes.io/infra": ""},
			Tolerations: []corev1.Toleration{{
				Key:      "node-role.kubernetes.io/infra",
				Operator: "Exists",
			}}}).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create clusterLogging instance %s in namespace %s; %v",
		vcoreparams.CLOInstanceName, vcoreparams.CLONamespace, err))

	glog.V(100).Infof("Check clusterLogging pods")

	podsList, err := pod.ListByNamePattern(APIClient, vcoreparams.CLOName, vcoreparams.CLONamespace)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
		vcoreparams.CLOName, vcoreparams.CLONamespace, err))
	Expect(len(podsList)).To(Equal(1), fmt.Sprintf("pod %s not found in namespace %s",
		vcoreparams.CLOName, vcoreparams.CLONamespace))

	err = podsList[0].WaitUntilReady(time.Second)
	Expect(err).ToNot(HaveOccurred(), "pod %s in namespace %s is not ready",
		podsList[0].Definition.Name, vcoreparams.CLONamespace)

	err = podsList[0].WaitUntilRunning(5 * time.Second)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
		podsList[0].Definition.Name, vcoreparams.CLONamespace, err))

	odfMcp := mco.NewMCPBuilder(APIClient, VCoreConfig.OdfMCPName)
	if odfMcp.IsInCondition("Updated") {
		podsList, err := pod.ListByNamePattern(APIClient, loggingCollectorPodNamePattern, vcoreparams.CLONamespace)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
			loggingCollectorPodNamePattern, vcoreparams.CLONamespace, err))
		Expect(len(podsList)).To(Equal(10), fmt.Sprintf("not all pods %s found in namespace %s: %v",
			loggingCollectorPodNamePattern, vcoreparams.CLONamespace, podsList))

		for _, pod := range podsList {
			err = pod.WaitUntilReady(time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s is not ready",
				pod.Definition.Name, vcoreparams.CLONamespace))

			err = pod.WaitUntilRunning(5 * time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
				pod.Definition.Name, vcoreparams.CLONamespace, err))
		}

		for _, clusterLoggingPodNamePattern := range loggingLokiOnePodNamePattern {
			podsList, err = pod.ListByNamePattern(APIClient, clusterLoggingPodNamePattern, vcoreparams.CLONamespace)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
				clusterLoggingPodNamePattern, vcoreparams.CLONamespace, err))
			Expect(len(podsList)).To(Equal(1), fmt.Sprintf("pods %s not found in namespace %s",
				clusterLoggingPodNamePattern, vcoreparams.CLONamespace))

			err = podsList[0].WaitUntilReady(time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s is not ready",
				podsList[0].Definition.Name, vcoreparams.CLONamespace))

			err = podsList[0].WaitUntilRunning(5 * time.Second)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
				podsList[0].Definition.Name, vcoreparams.CLONamespace, err))
		}

		for _, clusterLoggingPodNamePattern := range loggingLokiTwoPodsNamePattern {
			podsList, err = pod.ListByNamePattern(APIClient, clusterLoggingPodNamePattern, vcoreparams.CLONamespace)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s not found in namespace %s; %v",
				clusterLoggingPodNamePattern, vcoreparams.CLONamespace, err))
			Expect(len(podsList)).To(Equal(2), fmt.Sprintf("pods %s not found in namespace %s",
				clusterLoggingPodNamePattern, vcoreparams.CLONamespace))

			for _, pod := range podsList {
				err = pod.WaitUntilReady(time.Second)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s is not ready",
					pod.Definition.Name, vcoreparams.CLONamespace))

				err = pod.WaitUntilRunning(5 * time.Second)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pod %s in namespace %s failed to run; %v",
					pod.Definition.Name, vcoreparams.CLONamespace, err))
			}
		}
	}

	glog.V(vcoreparams.VCoreLogLevel).Infof("Enable logging-view-plugin")

	consoleoperatorObj, err := console.PullConsoleOperator(APIClient, "cluster")
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("consoleoperator is unavailable: %v", err))

	_, err = consoleoperatorObj.WithPlugins([]string{"logging-view-plugin"}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to enable logging-view-pluggin due to %v", err))

	_, err = consoleoperatorObj.WithPlugins([]string{"logging-view-plugin"}, false).Update()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to enable logging-view-pluggin due to %v", err))
} // func CreateCLOInstance (ctx SpecContext)
