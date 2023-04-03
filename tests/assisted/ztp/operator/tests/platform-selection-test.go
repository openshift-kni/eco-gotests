package operator_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-gotests/pkg/assisted"
	"github.com/openshift-kni/eco-gotests/pkg/hive"
	"github.com/openshift-kni/eco-gotests/pkg/namespace"
	"github.com/openshift-kni/eco-gotests/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	platformtypeSpoke = "platform-test"
)

var (
	testNS                  *namespace.Builder
	testSecret              *secret.Builder
	testClusterDeployment   *hive.ClusterDeploymentBuilder
	testAgentClusterInstall *assisted.AgentClusterInstallBuilder
	err                     error
)

var _ = Describe(
	"PlatformSelection",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelPlatformSelectionTestCases), func() {
		When("on ACM 2.6", func() {
			BeforeAll(func() {
				By("Check that hub meets operator version requirement")
				reqMet, msg := meets.HubOperatorVersionRequirement("2.6")
				if !reqMet {
					Skip(msg)
				}

				By("Check that spoke meets ocp version requirement")
				reqMet, msg = meets.SpokeOCPVersionRequirement("4.8")
				if !reqMet {
					Skip(msg)
				}

				By("Check that spoke meets pull-secret requirement")
				reqMet, msg = meets.SpokePullSecretSetRequirement()
				if !reqMet {
					Skip(msg)
				}

				tsparams.ReporterNamespacesToDump[platformtypeSpoke] = "platform-selection namespace"

				By("Create platform-test namespace")
				testNS, err = namespace.NewBuilder(APIClient, platformtypeSpoke).Create()
				Expect(err).ToNot(HaveOccurred(), "error occurred when creating namespace")

				By("Create platform-test pull-secret")
				testSecret, err = secret.NewBuilder(
					APIClient,
					fmt.Sprintf("%s-pull-secret", platformtypeSpoke),
					platformtypeSpoke,
					v1.SecretTypeDockerConfigJson).WithData(map[string][]byte{
					".dockerconfigjson": []byte(ZTPConfig.SpokeConfig.PullSecret)}).Create()
				Expect(err).ToNot(HaveOccurred(), "error occurred when creating pull-secret")

				By("Create platform-test clusterdeployment")
				testClusterDeployment, err = hive.NewABMClusterDeploymentBuilder(
					APIClient,
					platformtypeSpoke,
					testNS.Definition.Name,
					platformtypeSpoke,
					"qe.lab.redhat.com",
					platformtypeSpoke,
					metaV1.LabelSelector{
						MatchLabels: map[string]string{
							"dummy": "label",
						},
					}).WithPullSecret(testSecret.Definition.Name).Create()
				Expect(err).ToNot(HaveOccurred(), "error occurred when creating clusterdeployment")
			})

			DescribeTable("defining agentclusterinstall",
				func(
					platformType v1beta1.PlatformType,
					userManagedNetworking bool,
					masterCount int,
					workerCount int,
					message string) {

					By("Create agentclusterinstall")
					testAgentClusterInstall, err = assisted.NewAgentClusterInstallBuilder(
						APIClient,
						platformtypeSpoke,
						testNS.Definition.Name,
						testClusterDeployment.Definition.Name,
						masterCount,
						workerCount,
						v1beta1.Networking{
							ClusterNetwork: []v1beta1.ClusterNetworkEntry{{
								CIDR:       "10.128.0.0/14",
								HostPrefix: 23,
							}},
							MachineNetwork: []v1beta1.MachineNetworkEntry{{
								CIDR: "192.168.254.0/24",
							}},
							ServiceNetwork: []string{"172.30.0.0/16"},
						}).WithImageSet(ZTPConfig.SpokeConfig.OCPVersion).
						WithPlatformType(platformType).
						WithUserManagedNetworking(userManagedNetworking).Create()
					Expect(err).ToNot(HaveOccurred(), "error creating agentclusterinstall")

					By("getting SpecSynced condition")
					specSyncedCondition, err := testAgentClusterInstall.GetCondition(v1beta1.ClusterSpecSyncedCondition)
					Expect(err).ToNot(HaveOccurred(), "error getting SpecSynced condition from agentclusterinstall")

					By("waiting for condition to report expected failure message")
					_, err = testAgentClusterInstall.WaitForConditionMessage(
						specSyncedCondition,
						"The Spec could not be synced due to an input error: "+message,
						time.Second*10)
					Expect(specSyncedCondition.Message).To(
						Equal("The Spec could not be synced due to an input error: "+message),
						"got unexpected message from SpecSynced condition")
					Expect(err).ToNot(HaveOccurred(), "error waiting for condition message")
				},
				Entry("that is SNO with VSphere platform", v1beta1.VSpherePlatformType, true, 1, 0,
					"Single node cluster is not supported alongside vsphere platform", polarion.ID("56198")),
				Entry("that is BareMetal platform with user-managed-networking", v1beta1.BareMetalPlatformType, true, 3, 2,
					"Can't set baremetal platform with user-managed-networking enabled", polarion.ID("56418")),
				Entry("that is None platform without user-managed-networking", v1beta1.NonePlatformType, false, 3, 2,
					"Can't set none platform with user-managed-networking disabled", polarion.ID("56420")),
			)

			AfterEach(func() {
				By("Delete agentclusterinstall")
				err := testAgentClusterInstall.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting agentclusterinstall")
			})

			AfterAll(func() {
				By("Delete platform-test clusterdeployment")
				_, err := testClusterDeployment.Delete()
				Expect(err).ToNot(HaveOccurred(), "error occurred when deleting clusterdeployment")

				By("Delete platform-test pull-secret")
				Expect(testSecret.Delete()).ToNot(HaveOccurred(), "error occurred when deleting pull-secret")

				By("Delete platform-test namespace")
				Expect(testNS.Delete()).ToNot(HaveOccurred(), "error occurred when deleting namespace")
			})
		})
	})
