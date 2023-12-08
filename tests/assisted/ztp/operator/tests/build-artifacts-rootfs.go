package operator_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cavaliergopher/cpio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-gotests/tests/assisted/internal/url"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/meets"
	. "github.com/openshift-kni/eco-gotests/tests/assisted/ztp/internal/ztpinittools"
	"github.com/openshift-kni/eco-gotests/tests/assisted/ztp/operator/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	"github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	rootfsTestName    = "rootfs-test"
	rootfsDownloadDir = "/tmp/assistedrootfs"
)

var (
	rootfsTestNS                  *namespace.Builder
	rootfsTestPullSecret          *secret.Builder
	rootfsTestClusterDeployment   *hive.ClusterDeploymentBuilder
	rootfsTestAgentClusterInstall *assisted.AgentClusterInstallBuilder
	rootfsTestInfraEnv            *assisted.InfraEnvBuilder
	rootfsISOId, rootfsImgID      string
)

var _ = Describe(
	"RootFS build artifact can be constructed from discovery ISO",
	Ordered,
	ContinueOnFailure,
	Label(tsparams.LabelBuildArtifcatRootFSTestCases), func() {
		When("on MCE 2.1 and above", func() {
			BeforeAll(func() {
				By("Check that clusterimageset exists")
				_, err = hive.PullClusterImageSet(HubAPIClient, ZTPConfig.HubOCPXYVersion)
				if err != nil {
					Skip(ZTPConfig.HubOCPXYVersion + " clusterimageset not present")
				}

				By("Check that rootfs was not passed via osImages")
				for _, osImage := range ZTPConfig.HubAgentServiceConfg.Object.Spec.OSImages {
					if osImage.OpenshiftVersion == ZTPConfig.HubOCPXYVersion && osImage.RootFSUrl != "" {
						Skip("RootFSUrl was provided through osImages")
					}
				}

				By("Create test namespace")
				rootfsTestNS, err = namespace.NewBuilder(HubAPIClient, rootfsTestName).Create()
				Expect(err).ToNot(HaveOccurred(), "error creating %s namespace", rootfsTestName)

				By("Create test pull-secret")
				rootfsTestPullSecret, err = secret.NewBuilder(
					HubAPIClient,
					rootfsTestName,
					rootfsTestName,
					v1.SecretTypeDockerConfigJson).WithData(ZTPConfig.HubPullSecret.Definition.Data).Create()
				Expect(err).ToNot(HaveOccurred(), "error creating %s pull-secret", rootfsTestName)

				By("Create test clusterdeployment")
				rootfsTestClusterDeployment, err = hive.NewABMClusterDeploymentBuilder(HubAPIClient,
					rootfsTestName,
					rootfsTestName,
					rootfsTestName,
					"qe.redhat.com",
					rootfsTestName,
					metaV1.LabelSelector{MatchLabels: map[string]string{
						"dummy": "label",
					}}).WithPullSecret(rootfsTestName).Create()

				Expect(err).ToNot(HaveOccurred(), "error creating %s clusterdeployment", rootfsTestName)

				By("Create test agentclusterinstall")
				rootfsTestAgentClusterInstall, err = assisted.NewAgentClusterInstallBuilder(HubAPIClient,
					rootfsTestName,
					rootfsTestName,
					rootfsTestName,
					3,
					2,
					v1beta1.Networking{ClusterNetwork: []v1beta1.ClusterNetworkEntry{{
						CIDR:       "10.128.0.0/14",
						HostPrefix: 23,
					}},
						MachineNetwork: []v1beta1.MachineNetworkEntry{{
							CIDR: "192.168.254.0/24",
						}},
						ServiceNetwork: []string{"172.30.0.0/16"}}).WithImageSet(ZTPConfig.HubOCPXYVersion).Create()

				Expect(err).ToNot(HaveOccurred(), "error creating %s agentclusterinstall", rootfsTestName)

				By("Create test infraenv")
				rootfsTestInfraEnv, err = assisted.NewInfraEnvBuilder(
					HubAPIClient,
					rootfsTestName,
					rootfsTestName,
					rootfsTestName).WithClusterRef(rootfsTestName, rootfsTestName).Create()
				Expect(err).ToNot(HaveOccurred(), "error creating %s infraenv", rootfsTestName)

				_, err = rootfsTestInfraEnv.WaitForDiscoveryISOCreation(time.Minute * 3)
				Expect(err).ToNot(HaveOccurred(), "error waiting for download url to be created")

				if _, err = os.Stat(rootfsDownloadDir); err != nil {
					err = os.RemoveAll(rootfsDownloadDir)
					Expect(err).ToNot(HaveOccurred(), "error removing existing downloads directory")
				}

				err = os.Mkdir(rootfsDownloadDir, 0755)
				Expect(err).ToNot(HaveOccurred(), "error creating downloads directory")

				err = url.DownloadToDir(rootfsTestInfraEnv.Object.Status.ISODownloadURL, rootfsDownloadDir, true)
				Expect(err).ToNot(HaveOccurred(), "error downloading ISO")

				err = url.DownloadToDir(rootfsTestInfraEnv.Object.Status.BootArtifacts.RootfsURL, rootfsDownloadDir, true)
				Expect(err).ToNot(HaveOccurred(), "error downloading rootfs")

				dirEntry, err := os.ReadDir(rootfsDownloadDir)
				Expect(err).ToNot(HaveOccurred(), "error reading in downloads dir")

				var isoFile string
				var rootfsFile string

				for _, entry := range dirEntry {
					if strings.Contains(entry.Name(), ".iso") {
						isoFile = entry.Name()
					}

					if strings.Contains(entry.Name(), ".img") {
						rootfsFile = entry.Name()
					}
				}

				Expect(isoFile).NotTo(BeEmpty(), "error finding discovery iso")
				Expect(rootfsFile).NotTo(BeEmpty(), "error finding rootfs img")

				rootfsISOId, err = getISOVolumeID(fmt.Sprintf("%s/%s", rootfsDownloadDir, isoFile))
				Expect(err).NotTo(HaveOccurred(), "error reading volume id from iso")

				rootfsImgID, err = getCoreOSRootfsValue(fmt.Sprintf("%s/%s", rootfsDownloadDir, rootfsFile))
				Expect(err).NotTo(HaveOccurred(), "error reading coreos-live-rootfs from rootfs img")
			})

			DescribeTable("Rootfs is correctly extracted", func(requirement func() (bool, string)) {
				if reqMet, msg := requirement(); !reqMet {
					Skip(msg)
				}

				Expect(rootfsISOId).To(Equal(rootfsImgID), "error: iso and rootfs ids do not match")
			},
				Entry("in connected environments", meets.HubConnectedRequirement, polarion.ID("53721")),
				Entry("in disconnected environments", meets.HubDisconnectedRequirement, polarion.ID("53722")),
			)

			AfterAll(func() {
				err = rootfsTestInfraEnv.Delete()
				Expect(err).ToNot(HaveOccurred(), "error removing infraenv")

				err = rootfsTestAgentClusterInstall.Delete()
				Expect(err).ToNot(HaveOccurred(), "error removing agentclusterinstall")

				err = rootfsTestClusterDeployment.Delete()
				Expect(err).ToNot(HaveOccurred(), "error removing clusterdeployment")

				err = rootfsTestPullSecret.Delete()
				Expect(err).ToNot(HaveOccurred(), "error removing pull-secret")

				err = rootfsTestNS.Delete()
				Expect(err).ToNot(HaveOccurred(), "error removing namespace")

				err = os.RemoveAll(rootfsDownloadDir)
				Expect(err).ToNot(HaveOccurred(), "error removing download dir")
			})
		})
	})

func getISOVolumeID(path string) (string, error) {
	iso, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer iso.Close()

	// The first 32768 bytes are unused by the ISO 9660 standard, typically for bootable media
	// This is where the data area begins and the 32 byte string representing the volume identifier
	// is offset 40 bytes into the primary volume descriptor
	// Ref: https://github.com/openshift/assisted-image-service/blob/main/pkg/isoeditor/isoutil.go#L208
	volID := make([]byte, 32)
	_, err = iso.ReadAt(volID, 32808)

	if err != nil {
		return "", err
	}

	strVolID := string(volID)
	strVolID = strings.TrimLeft(strVolID, "rhcos-")
	strVolID = strings.ReplaceAll(strVolID, "\x00", "")

	return strVolID, nil
}

func getCoreOSRootfsValue(path string) (string, error) {
	img, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer img.Close()
	reader := cpio.NewReader(img)

	for hdr, err := reader.Next(); !errors.Is(err, io.EOF); hdr, err = reader.Next() {
		if err != nil {
			return "", err
		}

		if hdr.Name == "etc/coreos-live-rootfs" {
			content := make([]byte, hdr.Size)

			_, err = reader.Read(content)
			if err != nil {
				return "", err
			}

			return strings.TrimSpace(string(content)), nil
		}
	}

	return "", fmt.Errorf("error finding etc/coreos-live-rootfs from %s", path)
}
