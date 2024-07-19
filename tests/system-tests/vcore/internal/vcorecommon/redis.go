package vcorecommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/lso"
	lsov1 "github.com/openshift/local-storage-operator/api/v1"
	lsov1alpha1 "github.com/openshift/local-storage-operator/api/v1alpha1"

	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"

	"github.com/openshift-kni/eco-goinfra/pkg/statefulset"

	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/mirroring"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/template"
	corev1 "k8s.io/api/core/v1"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

// VerifyRedisSuite container that contains tests for the Redis deployment verification.
func VerifyRedisSuite() {
	Describe(
		"Redis validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			BeforeAll(func() {
				By(fmt.Sprintf("Asserting %s folder exists", vcoreparams.ConfigurationFolderName))

				vcoreConfigsFolder := filepath.Join(VCoreConfig.HomeDir, vcoreparams.ConfigurationFolderName)

				if err := os.Mkdir(vcoreConfigsFolder, 0755); os.IsExist(err) {
					glog.V(vcoreparams.VCoreLogLevel).Infof("%s folder already exists", vcoreConfigsFolder)
				}
			})

			It("Verify redis localvolumeset instance exists",
				Label("redis"), VerifyRedisLocalVolumeSet)

			It("Verify Redis deployment procedure",
				Label("redis"), reportxml.ID("59503"), VerifyRedisDeploymentProcedure)
		})
}

// VerifyRedisLocalVolumeSet asserts redis localvolumeset instance exists.
func VerifyRedisLocalVolumeSet(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Create redis localvolumeset instance %s in namespace %s if not found",
		vcoreparams.RedisLocalVolumeSetName, vcoreparams.LSONamespace)

	var err error

	localVolumeSetObj := lso.NewLocalVolumeSetBuilder(APIClient,
		vcoreparams.RedisLocalVolumeSetName,
		vcoreparams.LSONamespace)

	if localVolumeSetObj.Exists() {
		err = localVolumeSetObj.Delete()
		Expect(err).ToNot(HaveOccurred(),
			fmt.Sprintf("failed to delete localvolumeset %s from namespace %s; %v",
				vcoreparams.RedisLocalVolumeSetName, vcoreparams.LSONamespace, err))
	}

	nodeSelector := corev1.NodeSelector{NodeSelectorTerms: []corev1.NodeSelectorTerm{{
		MatchExpressions: []corev1.NodeSelectorRequirement{{
			Key:      "cluster.ocs.openshift.io/openshift-storage",
			Operator: "In",
			Values:   []string{""},
		}}},
	}}

	deviceInclusionSpec := lsov1alpha1.DeviceInclusionSpec{
		DeviceTypes:                []lsov1alpha1.DeviceType{lsov1alpha1.RawDisk},
		DeviceMechanicalProperties: []lsov1alpha1.DeviceMechanicalProperty{lsov1alpha1.NonRotational},
	}

	_, err = localVolumeSetObj.WithNodeSelector(nodeSelector).
		WithStorageClassName(vcoreparams.RedisStorageClassName).
		WithVolumeMode(lsov1.PersistentVolumeBlock).
		WithFSType("ext4").
		WithMaxDeviceCount(int32(10)).
		WithDeviceInclusionSpec(deviceInclusionSpec).Create()
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create localvolumeset %s in namespace %s "+
		"due to %v", vcoreparams.RedisLocalVolumeSetName, vcoreparams.LSONamespace, err))
} // func VerifyLocalVolumeSet (ctx SpecContext)

// VerifyRedisDeploymentProcedure asserts Redis deployment procedure.
//
//nolint:funlen
func VerifyRedisDeploymentProcedure(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify Redis can be installed and works correctly")

	redisAppName := "redis-ha"
	redisNamespace := "redis-ha"
	redisSecretName := "redis-secret"
	redisCustomValuesTemplate := "redis-custom-values.yaml"
	redisStatefulsetName := "redis-ha-server"

	glog.V(vcoreparams.VCoreLogLevel).Info("Check if redis already installed")

	redisStatefulset, err := statefulset.Pull(APIClient, redisStatefulsetName, redisNamespace)

	if err == nil && redisStatefulset.IsReady(time.Second) {
		glog.V(vcoreparams.VCoreLogLevel).Infof("redis statefulset %s in namespace %s exists and ready",
			redisStatefulsetName, redisNamespace)
	} else {
		vcoreConfigsFolder := filepath.Join(VCoreConfig.HomeDir, vcoreparams.ConfigurationFolderName)

		redisConfigFilePath := filepath.Join(vcoreConfigsFolder, redisCustomValuesTemplate)

		redisImageRepository := "quay.io/cloud-bulldozer"
		redisImageName := "redis"
		redisImageTag := "latest"

		glog.V(vcoreparams.VCoreLogLevel).Info("Check that cluster pull-secret can be retrieved")

		clusterPullSecret, err := cluster.GetOCPPullSecret(APIClient)
		Expect(err).ToNot(HaveOccurred(), "error occurred when retrieving cluster pull-secret")

		imageURL := fmt.Sprintf("%s/%s", redisImageRepository, redisImageName)

		isDisconnected, err := platform.IsDisconnectedDeployment(APIClient)
		Expect(err).ToNot(HaveOccurred(), "failed to detect a deployment type")

		if !isDisconnected {
			glog.V(vcoreparams.VCoreLogLevel).Info("The connected deployment type was detected, " +
				"the images mirroring is not required")
		} else {
			glog.V(vcoreparams.VCoreLogLevel).Info("Mirror redis image locally")

			imageURL, _, err = mirroring.MirrorImageToTheLocalRegistry(
				APIClient,
				redisImageRepository,
				redisImageName,
				redisImageTag,
				VCoreConfig.Host,
				VCoreConfig.User,
				VCoreConfig.Pass,
				VCoreConfig.CombinedPullSecretFile,
				VCoreConfig.RegistryRepository,
				VCoreConfig.HomeDir)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to mirror redis image locally due to %v", err))
		}

		glog.V(vcoreparams.VCoreLogLevel).Infof("Install redis")

		installRedisCmd := []string{
			"helm repo add dandydev https://dandydeveloper.github.io/charts",
			"helm repo update",
			fmt.Sprintf("helm fetch dandydev/redis-ha --version 4.12.9 -d %s/.", vcoreConfigsFolder),
			fmt.Sprintf("tar xvfz %s/redis-ha-4.12.9.tgz --directory=%s/.",
				vcoreConfigsFolder, VCoreConfig.HomeDir)}
		for _, cmd := range installRedisCmd {
			_, err = shell.ExecuteCmd(cmd)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s command due to %v", cmd, err))
		}

		glog.V(vcoreparams.VCoreLogLevel).Info("Create redis namespace")

		redisNamespaceBuilder := namespace.NewBuilder(APIClient, redisNamespace)

		if !redisNamespaceBuilder.Exists() {
			_, err = redisNamespaceBuilder.Create()
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create namespace %s due to %v",
				redisNamespace, err))
			Expect(redisNamespaceBuilder.Exists()).To(Equal(true),
				fmt.Sprintf("namespace %s not found", redisNamespace))
		}

		glog.V(vcoreparams.VCoreLogLevel).Infof("Create redis secret %s in namespace %s",
			redisSecretName, redisNamespace)

		redisSecretBuilder := secret.NewBuilder(
			APIClient,
			redisSecretName,
			redisNamespace,
			corev1.SecretTypeDockerConfigJson)

		if redisSecretBuilder.Exists() {
			err = redisSecretBuilder.Delete()
			Expect(err).ToNot(HaveOccurred(), "failed to delete redis secret %s in namespace %s due to %w",
				redisSecretName, redisNamespace, err)
		}

		redisSecretBuilder, err = redisSecretBuilder.WithData(clusterPullSecret.Object.Data).Create()
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to create redis secret %s in namespace %s due %v",
			redisSecretName, redisNamespace, err))
		Expect(redisSecretBuilder.Exists()).To(Equal(true),
			fmt.Sprintf("Failed to create redis secret %s in namespace %s", redisSecretName, redisNamespace))

		glog.V(vcoreparams.VCoreLogLevel).Info("Get runAsUser and fsGroup values")

		fsGroupFull := redisNamespaceBuilder.Object.Annotations["openshift.io/sa.scc.supplemental-groups"]
		Expect(fsGroupFull).ToNot(Equal(""), fmt.Sprintf("failed to get fsGroup value for the namespase %s",
			redisNamespace))

		fsGroup := strings.Split(fsGroupFull, "/")[0]

		runAsUserFull := redisNamespaceBuilder.Object.Annotations["openshift.io/sa.scc.uid-range"]
		Expect(runAsUserFull).ToNot(Equal(""), fmt.Sprintf("failed to get runAsUser value for the namespase %s",
			redisNamespace))

		runAsUser := strings.Split(runAsUserFull, "/")[0]

		glog.V(vcoreparams.VCoreLogLevel).Info("Redis custom config")

		varsToReplace := make(map[string]interface{})
		varsToReplace["ImageRepository"] = imageURL
		varsToReplace["ImageTag"] = redisImageTag
		varsToReplace["RedisSecret"] = redisSecretName
		varsToReplace["StorageClass"] = vcoreparams.ODFStorageClassName
		// varsToReplace["StorageClass"] = "ocs-storagecluster-cephfs"
		varsToReplace["RunAsUser"] = runAsUser
		varsToReplace["FsGroup"] = fsGroup

		destinationDirectoryPath := filepath.Join(VCoreConfig.HomeDir, vcoreparams.ConfigurationFolderName)

		workingDir, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred(), err)

		templateDir := filepath.Join(workingDir, vcoreparams.TemplateFilesFolder)

		err = template.SaveTemplate(templateDir, redisCustomValuesTemplate, destinationDirectoryPath,
			redisCustomValuesTemplate, varsToReplace)
		Expect(err).ToNot(HaveOccurred(), "failed to create config file %s at folder %s due to %w",
			redisCustomValuesTemplate, vcoreConfigsFolder, err)

		customConfigCmd := fmt.Sprintf("helm upgrade --install %s -n %s %s/%s -f %s --kubeconfig %s",
			redisAppName, redisNamespace, VCoreConfig.HomeDir, redisAppName,
			redisConfigFilePath, os.Getenv("KUBECONFIG"))
		glog.V(vcoreparams.VCoreLogLevel).Infof("Execute command %s", customConfigCmd)

		result, err := shell.ExecuteCmd(customConfigCmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to config redis due to %v", err))
		Expect(strings.Contains(string(result), "STATUS: deployed")).To(Equal(true),
			fmt.Sprintf("redis is not properly configured: %s", string(result)))
	}

	odfMcp := mco.NewMCPBuilder(APIClient, VCoreConfig.OdfMCPName)
	if odfMcp.Exists() {
		glog.V(vcoreparams.VCoreLogLevel).Info("Wait for the statefulset ready")

		redisStatefulset, err := statefulset.Pull(APIClient, redisStatefulsetName, redisNamespace)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("statefulset %s not found in namespace %s; %v",
			redisStatefulsetName, redisNamespace, err))
		Expect(redisStatefulset.IsReady(5*time.Minute)).To(Equal(true),
			fmt.Sprintf("statefulset %s in namespace %s is not ready after 5 minutes",
				redisStatefulsetName, redisNamespace))

		glog.V(vcoreparams.VCoreLogLevel).Info("Verify redis server pods count")

		podsList, err := pod.ListByNamePattern(APIClient, redisAppName, redisNamespace)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("pods %s not found in namespace %s; %v",
			redisAppName, redisNamespace, err))
		Expect(len(podsList)).To(Equal(3), fmt.Sprintf("not all redis servers pods %s found in namespace %s;"+
			"expected: 3, found: %d", redisAppName, redisNamespace, len(podsList)))
	}
} // func VerifyRedisDeploymentProcedure (ctx SpecContext)
