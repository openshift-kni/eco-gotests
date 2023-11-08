package operator_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/assisted/internal/url"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/find"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpparams"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	agentInstallV1Beta1 "github.com/openshift/assisted-service/api/v1beta1"
	"github.com/openshift/assisted-service/models"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infraenvTestSpoke string = "infraenv-spoke"
)

var (
	hubClusterVersion                                          string
	testClusterImageSetName                                    string
	nsBuilder                                                  *namespace.Builder
	archURL                                                    string
	platformType                                               v1beta1.PlatformType
	userManagedNetworking                                      bool
	originalClusterImagesetBuilder, tempClusterImagesetBuilder *hive.ClusterImageSetBuilder
	mirrorRegistryRef                                          *v1.LocalObjectReference
)
var _ = Describe(
	"MultiArchitectureImage",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelMultiArchitectureImageTestCases), func() {
		When("on MCE 2.2 and above", func() {
			BeforeAll(func() {
				By("Check the hub cluster is connected")
				connectedCluster, _ := meets.HubConnectedRequirement()
				if !connectedCluster {
					Skip("The hub cluster must be connected")
				}

				By("Check that hub pull-secret can be retrieved")
				hubPullSecret, err = cluster.GetOCPPullSecret(HubAPIClient)
				Expect(err).ToNot(HaveOccurred(), "error occurred when retrieving hub pull-secret")

				nsBuilder = namespace.NewBuilder(HubAPIClient, infraenvTestSpoke)

				By("Retrieve the pre-existing AgentServiceConfig")
				var err error
				agentServiceConfigBuilder, err = assisted.PullAgentServiceConfig(HubAPIClient)
				Expect(err).ShouldNot(HaveOccurred(), "failed to pull AgentServiceConfig.")

				By("Grab the MirrorRegistryRef value from the spec")
				mirrorRegistryRef = agentServiceConfigBuilder.Definition.Spec.MirrorRegistryRef

				By("Delete the pre-existing AgentServiceConfig")
				err = agentServiceConfigBuilder.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting pre-existing agentserviceconfig")

				By("Get the OCP cluster version")
				hubClusterVersion, err = find.HubClusterVersionXY()
				Expect(err).ToNot(HaveOccurred(), "error retrieving the cluster version")
				glog.V(ztpparams.ZTPLogLevel).Infof("The cluster version is %s", hubClusterVersion)

				By("Retrieve ClusterImageSet before all tests if exists")
				testClusterImageSetName = hubClusterVersion + "-test"
				originalClusterImagesetBuilder, err = hive.PullClusterImageSet(HubAPIClient, testClusterImageSetName)

				By("Delete ClusterImageSet if exists before all tests ")
				if err == nil {
					_, err = originalClusterImagesetBuilder.Delete()
					Expect(err).ToNot(HaveOccurred(), "error deleting clusterimageset %s", testClusterImageSetName)
					glog.V(ztpparams.ZTPLogLevel).Infof("The ClusterImageSet %s was deleted", testClusterImageSetName)
				} else {
					glog.V(ztpparams.ZTPLogLevel).Infof("The ClusterImageSet %s doesn't exist. No attempt to delete it",
						testClusterImageSetName)
				}

			})
			AfterEach(func() {
				By("Delete the temporary namespace after test")
				if nsBuilder.Exists() {
					err := nsBuilder.Delete()
					Expect(err).ToNot(HaveOccurred(), "error deleting the temporary namespace after test")
				}

				By("Delete ClusterImageSet after a test")
				_, err = tempClusterImagesetBuilder.Delete()
				Expect(err).ToNot(HaveOccurred(), "error deleting clusterimageset %s", testClusterImageSetName)
				glog.V(ztpparams.ZTPLogLevel).Infof("The ClusterImageSet %s was deleted", testClusterImageSetName)

				By("Delete AgentServiceConfig after test")
				err = tempAgentServiceConfigBuilder.DeleteAndWait(time.Second * 10)
				Expect(err).ToNot(HaveOccurred(), "error deleting agentserviceconfig after test")
			})
			AfterAll(func() {
				By("Re-create the original ClusterImageSet after all test if existed")
				if originalClusterImagesetBuilder != nil {
					_, err := originalClusterImagesetBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "error re-creating clusterimageset %s",
						testClusterImageSetName)
					glog.V(ztpparams.ZTPLogLevel).Infof("The ClusterImageSet %s was re-created", testClusterImageSetName)
				} else {
					glog.V(ztpparams.ZTPLogLevel).Infof("Skip on re-creating the ClusterImageSet after all tests - didn't exist.")
				}

				By("Re-create the original AgentServiceConfig after all tests")
				_, err = agentServiceConfigBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "error re-creating the original agentserviceconfig after all tests")

				_, err = agentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
				Expect(err).ToNot(HaveOccurred(),
					"error waiting until the original agentserviceconfig is deployed")

			})
			It("Assert valid ISO is created by InfraEnv with cpuArchitecture set to x86_64",
				polarion.ID("56183"), func() {

					By("Download the rhcos-live ISO")
					archURL = "https://mirror.openshift.com/pub/openshift-v4/amd64/dependencies/rhcos/" +
						hubClusterVersion + "/latest/rhcos-live.x86_64.iso"

					err = url.DownloadToDir(archURL, "/tmp")
					if err != nil {
						archURL = "https://mirror.openshift.com/pub/openshift-v4/amd64/dependencies/rhcos/pre-release/latest-" +
							hubClusterVersion + "/rhcos-live.x86_64.iso"
						err = url.DownloadToDir(archURL, "/tmp")
					}

					Expect(err).ToNot(HaveOccurred(), "error downloading %s", archURL)

					glog.V(ztpparams.ZTPLogLevel).Infof("Downloaded ISO from this URL: %s", archURL)

					By("Create AgentServiceConfig with the specific OSImage")

					tempAgentServiceConfigBuilder = assisted.NewDefaultAgentServiceConfigBuilder(HubAPIClient)

					if mirrorRegistryRef != nil {
						tempAgentServiceConfigBuilder.Definition.Spec.MirrorRegistryRef = mirrorRegistryRef
					}
					_, err = tempAgentServiceConfigBuilder.WithOSImage(agentInstallV1Beta1.OSImage{
						OpenshiftVersion: hubClusterVersion,
						Version:          hubClusterVersion,
						Url:              archURL,
						CPUArchitecture:  models.ClusterCPUArchitectureX8664}).Create()
					Expect(err).ToNot(HaveOccurred(),
						"error creating agentserviceconfig with the specific osimage")

					By("Wait until AgentServiceConfig with the specic OSImage is deployed")
					_, err = tempAgentServiceConfigBuilder.WaitUntilDeployed(time.Minute * 10)
					Expect(err).ToNot(HaveOccurred(),
						"error waiting until agentserviceconfig without imagestorage is deployed")

					By("Create ClusterImageSet")
					payloadURL := "https://openshift-release.apps.ci.l2s4.p1.openshiftapps.com/graph"
					payloadVersion, payloadImage, err := getLatestReleasePayload(payloadURL)
					Expect(err).ToNot(HaveOccurred(), "error getting latest release payload image")

					glog.V(ztpparams.ZTPLogLevel).Infof("ClusterImageSet %s will use version %s with image %s",
						testClusterImageSetName, payloadVersion, payloadImage)
					tempClusterImagesetBuilder, err = hive.NewClusterImageSetBuilder(
						HubAPIClient, testClusterImageSetName, payloadImage).Create()
					Expect(err).ToNot(HaveOccurred(), "error creating clusterimageset %s", testClusterImageSetName)

					createSpokeClusterResources()

					time.Sleep(time.Second * 20)
				})
		})
	})

// createSpokeClusterResources is a helper function that creates
// spoke cluster resources required for the test.
func createSpokeClusterResources() {
	By("Create namespace for the test")

	if nsBuilder.Exists() {
		glog.V(ztpparams.ZTPLogLevel).Infof("The namespace '%s' already exists",
			nsBuilder.Object.Name)
	} else {
		// create the namespace
		glog.V(ztpparams.ZTPLogLevel).Infof("Creating the namespace:  %v", infraenvTestSpoke)
		_, err := nsBuilder.Create()
		Expect(err).ToNot(HaveOccurred(), "error creating namespace '%s' :  %v ",
			nsBuilder.Definition.Name, err)
	}

	By("Create pull-secret in the new namespace")

	testSecret, err = secret.NewBuilder(
		HubAPIClient,
		fmt.Sprintf("%s-pull-secret", infraenvTestSpoke),
		infraenvTestSpoke,
		v1.SecretTypeDockerConfigJson).WithData(hubPullSecret.Object.Data).Create()
	Expect(err).ToNot(HaveOccurred(), "error occurred when creating pull-secret")

	By("Create clusterdeployment in the new namespace")

	testClusterDeployment, err = hive.NewABMClusterDeploymentBuilder(
		HubAPIClient,
		infraenvTestSpoke,
		infraenvTestSpoke,
		infraenvTestSpoke,
		"qe.lab.redhat.com",
		infraenvTestSpoke,
		metaV1.LabelSelector{
			MatchLabels: map[string]string{
				"dummy": "label",
			},
		}).WithPullSecret(testSecret.Definition.Name).Create()
	Expect(err).ToNot(HaveOccurred(), "error occurred when creating clusterdeployment")

	By("Create agentclusterinstall in the new namespace")

	testAgentClusterInstall, err = assisted.NewAgentClusterInstallBuilder(
		HubAPIClient,
		infraenvTestSpoke,
		infraenvTestSpoke,
		infraenvTestSpoke,
		3,
		2,
		v1beta1.Networking{
			ClusterNetwork: []v1beta1.ClusterNetworkEntry{{
				CIDR:       "10.128.0.0/14",
				HostPrefix: 23,
			}},
			MachineNetwork: []v1beta1.MachineNetworkEntry{{
				CIDR: "192.168.254.0/24",
			}},
			ServiceNetwork: []string{"172.30.0.0/16"},
		}).WithImageSet(testClusterImageSetName).
		WithPlatformType(platformType).
		WithUserManagedNetworking(userManagedNetworking).Create()
	Expect(err).ToNot(HaveOccurred(), "error creating agentclusterinstall")

	By("Create infraenv in the new namespace")

	_, err = assisted.NewInfraEnvBuilder(
		HubAPIClient,
		infraenvTestSpoke,
		infraenvTestSpoke,
		testSecret.Definition.Name).Create()
	Expect(err).ToNot(HaveOccurred(), "error creating infraenv")
}

// getLatestReleasePayload is a helper function that returns
// the latest release payload image for the version.
func getLatestReleasePayload(url string) (string, string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()

	urlContent, err := io.ReadAll(response.Body)
	if err != nil {
		return "", "", err
	}

	releaseImages := make(map[string]string)

	images, err := readJSON(urlContent)
	if err != nil {
		return "", "", err
	}

	for _, i := range images.Nodes {
		if strings.Contains(i.Version, hubClusterVersion) &&
			!strings.Contains(i.Version, "nightly") &&
			!strings.Contains(i.Payload, "dev-preview") {
			releaseImages[i.Version] = i.Payload
		}
	}

	releaseImageVersions := make([]string, 0)

	for version := range releaseImages {
		releaseImageVersions = append(releaseImageVersions, version)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(releaseImageVersions)))

	return releaseImageVersions[0], releaseImages[releaseImageVersions[0]], nil
}

// Images is a helper struct to encode OSImages correctly.
type Images struct {
	Nodes []struct {
		Version, Payload string
	} `json:"nodes"`
}

// readJSON is a helper function that reads a json from URL
// and returns the payload release version and image location.
func readJSON(urlContent []byte) (Images, error) {
	var images Images
	err := json.Unmarshal(urlContent, &images)

	if err != nil {
		return images, err
	}

	return images, nil
}