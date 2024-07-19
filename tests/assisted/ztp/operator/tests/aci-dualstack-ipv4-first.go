package operator_test

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/schemes/assisted/api/hiveextension/v1beta1"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpparams"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dualstackTestSpoke = "dualstacktest"
)

var _ = Describe(
	"DualstackIPv4First",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelDualstackIPv4FirstACI), func() {
		When("on MCE 2.0 and above", func() {
			BeforeAll(func() {

				By("Check that the ClusterImageSet exists")
				_, err := hive.PullClusterImageSet(HubAPIClient, ZTPConfig.HubOCPXYVersion)
				if err != nil {
					Skip("The ClusterImageSet must exist")
				}

				nsBuilder = namespace.NewBuilder(HubAPIClient, dualstackTestSpoke)
			})
			AfterEach(func() {
				By("Delete the temporary namespace after test")
				if nsBuilder.Exists() {
					err := nsBuilder.DeleteAndWait(time.Second * 300)
					Expect(err).ToNot(HaveOccurred(), "error deleting the temporary namespace after test")
				}
			})

			It("Validates that ACI with dualstack expects IPv4 first", reportxml.ID("44877"), func() {
				agentClusterInstallBuilder := createDualstackSpokeClusterResources()

				By("Waiting for specific error message from SpecSynced condition")
				err = agentClusterInstallBuilder.WaitForConditionMessage(v1beta1.ClusterSpecSyncedCondition,
					"The Spec could not be synced due to an input error: First machine network has to be IPv4 subnet", time.Second*10)
				Expect(err).NotTo(HaveOccurred(), "didn't get the expected message from SpecSynced condition")

			})

		})
	})

// createDualstackSpokeClusterResources is a helper function that creates
// spoke cluster resources required for the test.
func createDualstackSpokeClusterResources() *assisted.AgentClusterInstallBuilder {
	By("Create namespace for the test")

	if nsBuilder.Exists() {
		glog.V(ztpparams.ZTPLogLevel).Infof("The namespace '%s' already exists",
			nsBuilder.Object.Name)
	} else {
		// create the namespace
		glog.V(ztpparams.ZTPLogLevel).Infof("Creating the namespace:  %v", dualstackTestSpoke)

		_, err := nsBuilder.Create()
		Expect(err).ToNot(HaveOccurred(), "error creating namespace '%s' :  %v ",
			nsBuilder.Definition.Name, err)
	}

	By("Create pull-secret in the new namespace")

	testSecret, err := secret.NewBuilder(
		HubAPIClient,
		fmt.Sprintf("%s-pull-secret", dualstackTestSpoke),
		dualstackTestSpoke,
		corev1.SecretTypeDockerConfigJson).WithData(ZTPConfig.HubPullSecret.Object.Data).Create()
	Expect(err).ToNot(HaveOccurred(), "error occurred when creating pull-secret")

	By("Create clusterdeployment in the new namespace")

	testClusterDeployment, err := hive.NewABMClusterDeploymentBuilder(
		HubAPIClient,
		dualstackTestSpoke,
		dualstackTestSpoke,
		dualstackTestSpoke,
		"assisted.test.com",
		dualstackTestSpoke,
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				"dummy": "label",
			},
		}).WithPullSecret(testSecret.Definition.Name).Create()
	Expect(err).ToNot(HaveOccurred(), "error occurred when creating clusterdeployment")

	By("Create agentclusterinstall in the new namespace")

	agentClusterInstallBuilder, err := assisted.NewAgentClusterInstallBuilder(
		HubAPIClient,
		dualstackTestSpoke,
		dualstackTestSpoke,
		testClusterDeployment.Object.Name,
		3,
		2,
		v1beta1.Networking{
			ClusterNetwork: []v1beta1.ClusterNetworkEntry{
				{
					CIDR:       "fd01::/48",
					HostPrefix: 64,
				},
				{
					CIDR:       "11.128.0.0/14",
					HostPrefix: 24,
				},
			},
			MachineNetwork: []v1beta1.MachineNetworkEntry{
				{
					CIDR: "fd2e:6f44:5dd8:5::/64",
				},
				{
					CIDR: "192.168.254.0/24",
				},
			},
			ServiceNetwork: []string{"172.30.0.0/16", "fd02::/112"},
		}).WithImageSet(ZTPConfig.HubOCPXYVersion).WithAPIVip("192.168.254.5").
		WithIngressVip("192.168.254.10").Create()
	Expect(err).ToNot(HaveOccurred(), "error creating agentclusterinstall")

	return agentClusterInstallBuilder
}
