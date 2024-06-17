package vcorecommon

import (
	"fmt"
	kedav1alpha1 "github.com/kedacore/keda-olm-operator/apis/keda/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/keda"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/mirroring"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

const (
	kedaScaleObjectName       = "prometheus-scaledobject"
	configmapName             = "cluster-monitoring-config"
	configmapNamespace        = "openshift-monitoring"
	maxReplicaCount           = 8
	testAppServicemonitorName = "keda-testing-sm"
	serviceaccountName        = "thanos"
	triggerAuthName           = "keda-trigger-auth-prometheus"
	metricsReaderName         = "thanos-metrics-reader"

	prometheusOriginMirrorURL = "quay.io/zroubalik"
	prometheusImageName       = "prometheus-app"
	prometheusImageTag        = "latest"

	abOriginMirrorURL = "docker.io/jordi"
	abImageName       = "ab"
	abImageTag        = "latest"
)

// VerifyKedaNamespaceExists asserts namespace for NMState operator exists.
func VerifyKedaNamespaceExists(ctx SpecContext) {
	err := apiobjectshelper.VerifyNamespaceExists(APIClient, vcoreparams.KedaNamespace, time.Second)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to pull %q namespace", vcoreparams.KedaNamespace))
} // func VerifyKedaNamespaceExists (ctx SpecContext)

// VerifyKedaDeployment assert that Keda operator deployment succeeded.
func VerifyKedaDeployment(ctx SpecContext) {
	err := apiobjectshelper.VerifyOperatorDeployment(APIClient,
		vcoreparams.KedaSubscriptionName,
		vcoreparams.KedaDeploymentName,
		vcoreparams.KedaNamespace,
		time.Minute)
	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("Keda operator deployment %s failure in the namespace %s; %v",
			vcoreparams.KedaDeploymentName, vcoreparams.KedaNamespace, err))
} // func VerifyKedaDeployment (ctx SpecContext)

// VerifyKedaControllerDeployment assert that kedaController instance created successfully.
func VerifyKedaControllerDeployment(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify kedaController instance exists")

	kedaControllerBuilder := keda.NewKedaControllerBuilder(
		APIClient,
		vcoreparams.KedaControllerName,
		vcoreparams.KedaNamespace)

	if !kedaControllerBuilder.Exists() {
		var err error
		admissionWebhooks := kedav1alpha1.KedaAdmissionWebhooksSpec{
			LogLevel:   "info",
			LogEncoder: "console",
		}
		operator := kedav1alpha1.KedaOperatorSpec{
			LogLevel:   "info",
			LogEncoder: "console",
		}
		metricsServer := kedav1alpha1.KedaMetricsServerSpec{
			LogLevel: "0",
		}

		kedaControllerBuilder, err = kedaControllerBuilder.WithAdmissionWebhooks(admissionWebhooks).
			WithOperator(operator).
			WithMetricsServer(metricsServer).
			WithWatchNamespace(vcoreparams.KedaWatchNamespace).
			Create()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to create kedaController instance %q in namespace %q due to: %v",
				vcoreparams.KedaControllerName, vcoreparams.KedaNamespace, err.Error()))
		Expect(kedaControllerBuilder.Exists()).To(Equal(true), fmt.Sprintf(
			"no kedaController instance  %s was found in the namespace %s",
			vcoreparams.KedaControllerName, vcoreparams.KedaNamespace))
	}
} // func VerifyKedaControllerDeployment (ctx SpecContext)

// VerifyScaleObjectDeployment assert that scaleObject instance created successfully.
func VerifyScaleObjectDeployment(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Info("Verify monitoring status")
	var err error

	configMapBuilder := configmap.NewBuilder(APIClient, configmapName, configmapNamespace)

	if !configMapBuilder.Exists() {
		configMapBuilder, err = configMapBuilder.
			WithData(map[string]string{"config.yaml": "enableUserWorkload: true"}).Create()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to create configmap %q in namespace %q due to: %v",
				configmapName, configmapNamespace, err.Error()))
		Expect(configMapBuilder.Exists()).To(Equal(true), fmt.Sprintf(
			"no configmap %s was found in the namespace %s", configmapName, configmapNamespace))
	}

	glog.V(vcoreparams.VCoreLogLevel).Info("Deploy application that exposes Prometheus metrics")
	namespace

	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify scaleObject instance deployment and functionality")

	scaleObjectBuilder := keda.NewScaledObjectBuilder(APIClient, kedaScaleObjectName, vcoreparams.KedaWatchNamespace)
	if !scaleObjectBuilder.Exists() {
		var err error
		admissionWebhooks := kedav1alpha1.KedaAdmissionWebhooksSpec{
			LogLevel:   "info",
			LogEncoder: "console",
		}
		operator := kedav1alpha1.KedaOperatorSpec{
			LogLevel:   "info",
			LogEncoder: "console",
		}
		metricsServer := kedav1alpha1.KedaMetricsServerSpec{
			LogLevel: "0",
		}

		scaleObjectBuilder, err = scaleObjectBuilder.WithAdmissionWebhooks(admissionWebhooks).
			WithOperator(operator).
			WithMetricsServer(metricsServer).
			WithWatchNamespace(vcoreparams.KedaWatchNamespace).
			Create()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("Failed to create kedaController instance %q in namespace %q due to: %v",
				vcoreparams.KedaControllerName, vcoreparams.KedaNamespace, err.Error()))
		Expect(scaleObjectBuilder.Exists()).To(Equal(true), fmt.Sprintf(
			"no kedaController instance  %s was found in the namespace %s",
			vcoreparams.KedaControllerName, vcoreparams.KedaNamespace))
	}
} // func VerifyKedaControllerDeployment (ctx SpecContext)

// VerifyKedaSuite container that contains tests for Keda verification.
func VerifyKedaSuite() {
	Describe(
		"Keda validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			BeforeAll(func() {
				By(fmt.Sprintf("Asserting %s folder exists", vcoreparams.ConfigurationFolderName))

				homeDir, err := os.UserHomeDir()
				Expect(err).To(BeNil(), fmt.Sprint(err))

				vcoreConfigsFolder := filepath.Join(homeDir, vcoreparams.ConfigurationFolderName)

				glog.V(vcoreparams.VCoreLogLevel).Infof("vcoreConfigsFolder: %s", vcoreConfigsFolder)

				if err := os.Mkdir(vcoreConfigsFolder, 0755); os.IsExist(err) {
					glog.V(vcoreparams.VCoreLogLevel).Infof("%s folder already exists", vcoreConfigsFolder)
				}
			})

			It(fmt.Sprintf("Verifies %s namespace exists", vcoreparams.KedaNamespace),
				Label("keda"), VerifyKedaNamespaceExists)

			It("Verifies Keda operator deployment succeeded",
				Label("keda"), reportxml.ID("65001"), VerifyKedaDeployment)

			It("Verifies KedaController instance created successfully",
				Label("keda"), reportxml.ID("65004"), VerifyKedaControllerDeployment)

			It("Verifies ScaleObject instance created successfully",
				Label("keda"), reportxml.ID("65007"), VerifyScaleObjectDeployment)
		})
}

func getImageUrl(repository, name, tag string) (string, error) {
	imageURL := fmt.Sprintf("%s/%s", repository, name)

	isDisconnected, err := platform.IsDisconnectedDeployment(APIClient)

	if err != nil {
		return "", err
	}

	if !isDisconnected {
		glog.V(vcoreparams.VCoreLogLevel).Info("The connected deployment type was detected, " +
			"the images mirroring is not required")
	} else {
		glog.V(vcoreparams.VCoreLogLevel).Infof("Mirror image %s:%s locally", imageURL, tag)

		imageURL, _, err = mirroring.MirrorImageToTheLocalRegistry(
			APIClient,
			repository,
			name,
			tag,
			VCoreConfig.Host,
			VCoreConfig.User,
			VCoreConfig.Pass,
			VCoreConfig.CombinedPullSecretFile,
			VCoreConfig.RegistryRepository)

		if err != nil {
			return "", fmt.Errorf("failed to mirror image %s:%s locally due to %v",
				imageURL, tag, err)
		}
	}

	return imageURL, nil
}
