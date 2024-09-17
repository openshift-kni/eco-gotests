package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/bmh"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/hive"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/version"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ZTP BIOS Configuration Tests", Label(tsparams.LabelBiosDayZeroTests), func() {
	var (
		spokeClusterName string
		nodeNames        []string
	)

	// 75196 - Check if spoke has required BIOS setting values applied
	It("Check if SNO spoke has required BIOS setting values applied", reportxml.ID("75196"), func() {
		versionInRange, err := version.IsVersionStringInRange(RANConfig.ZTPVersion, "4.17", "")
		Expect(err).ToNot(HaveOccurred(), "Failed to check if ZTP version is in range")

		if !versionInRange {
			Skip("ZTP BIOS configuration tests require ZTP version of least 4.17")
		}

		spokeClusterName, err = GetSpokeClusterName(HubAPIClient, Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to get SNO cluster name")
		By(fmt.Sprintf("Cluster name: %s", spokeClusterName))

		nodeNames, err = GetNodeNames(Spoke1APIClient)
		Expect(err).ToNot(HaveOccurred(), "Failed to get node names")
		By(fmt.Sprintf("Node names: %v", nodeNames))

		By(fmt.Sprintf("cluster=%s SNO spoke=%s", spokeClusterName, nodeNames[0]))
		hfs, err := bmh.PullHFS(HubAPIClient, nodeNames[0], spokeClusterName)
		Expect(err).ToNot(
			HaveOccurred(),
			fmt.Sprintf("Failed to get HFS for spoke %s in cluster %s", nodeNames[0], spokeClusterName),
		)

		hfs_obj, err := hfs.Get()
		Expect(err).ToNot(
			HaveOccurred(),
			fmt.Sprintf("Failed to get HFS Obj for spoke %s in cluster %s", nodeNames[0], spokeClusterName),
		)

		hfs_req_settings := hfs_obj.Spec.Settings
		hfs_status_settings := hfs_obj.Status.Settings
		rc := true
		if len(hfs_req_settings) > 0 {
			Expect(len(hfs_status_settings) > 0).To(
				BeTrueBecause("hfs.spec.settings map is not empty, but hfs.status.settings map is empty"))

			for param, value := range hfs_req_settings {
				setting, ok := hfs_status_settings[param]
				if ok {
					req_setting := value.String()
					if req_setting == setting {
						By(fmt.Sprintf("Requested setting matches current: %s=%s", param, setting))
					} else {
						By(
							fmt.Sprintf("Requested setting %s value %s does not match current value %s", param, req_setting, setting))
						rc = false
					}
				} else {
					By(fmt.Sprintf("Current settings does not have param %s", param))
				}
			}

			Expect(rc).To(BeTrueBecause("One or more requested settings does not match current settings"))
		} else {
			Skip("hfs.spec.settings map is empty")
		}
	})

})

func GetSpokeClusterName(hubAPIClient, spokeAPIClient *clients.Settings) (string, error) {
	spokeClusterVersion, err := cluster.GetOCPClusterVersion(spokeAPIClient)
	if err != nil {
		return "", err
	}

	spokeClusterID := spokeClusterVersion.Object.Spec.ClusterID

	clusterDeployments, err := hive.ListClusterDeploymentsInAllNamespaces(hubAPIClient)
	if err != nil {
		return "", err
	}

	for _, clusterDeploymentBuilder := range clusterDeployments {
		if clusterDeploymentBuilder.Object.Spec.ClusterMetadata != nil &&
			clusterDeploymentBuilder.Object.Spec.ClusterMetadata.ClusterID == string(spokeClusterID) {
			return clusterDeploymentBuilder.Object.Spec.ClusterName, nil
		}
	}

	return "", fmt.Errorf("could not find ClusterDeployment from provided API clients")
}

func GetNodeNames(spokeAPIClient *clients.Settings) ([]string, error) {
	nodeList, err := nodes.List(
		spokeAPIClient,
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	nodeNames := []string{}
	for _, node := range nodeList {
		nodeNames = append(nodeNames, node.Definition.Name)
	}

	return nodeNames, nil
}
