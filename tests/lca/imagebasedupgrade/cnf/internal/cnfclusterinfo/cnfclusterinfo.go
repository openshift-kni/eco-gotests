package cnfclusterinfo

import (
	"errors"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfparams"
	"k8s.io/utils/strings/slices"
)

var (
	// PreUpgradeClusterInfo holds the cluster info pre upgrade.
	PreUpgradeClusterInfo = ClusterStruct{}

	// PostUpgradeClusterInfo holds the cluster info post upgrade.
	PostUpgradeClusterInfo = ClusterStruct{}
)

// ClusterStruct is a struct that holds the cluster version and id.
type ClusterStruct struct {
	Version   string
	ID        string
	Name      string
	Operators []string
	NodeName  string
}

// SaveClusterInfo is a dedicated func to save cluster info.
func (upgradeVar *ClusterStruct) SaveClusterInfo() error {
	clusterVersion, err := cluster.GetOCPClusterVersion(cnfinittools.TargetSNOAPIClient)

	if err != nil {
		glog.V(cnfparams.CNFLogLevel).Infof("Could not retrieve cluster version")

		return err
	}

	targetSnoClusterName, err := cluster.GetOCPClusterName(cnfinittools.TargetSNOAPIClient)

	if err != nil {
		glog.V(cnfparams.CNFLogLevel).Infof("Could not retrieve target sno cluster name")

		return err
	}

	csvList, err := olm.ListClusterServiceVersionInAllNamespaces(cnfinittools.TargetSNOAPIClient)

	if err != nil {
		glog.V(cnfparams.CNFLogLevel).Infof("Could not retrieve csv list")

		return err
	}

	var installedCSV []string

	for _, csv := range csvList {
		if !slices.Contains(installedCSV, csv.Object.Name) {
			installedCSV = append(installedCSV, csv.Object.Name)
		}
	}

	node, err := nodes.List(cnfinittools.TargetSNOAPIClient)

	if err != nil {
		glog.V(cnfparams.CNFLogLevel).Infof("Could not retrieve node list")

		return err
	}

	if len(node) == 0 {
		return errors.New("node list is empty")
	}

	upgradeVar.Version = clusterVersion.Object.Status.Desired.Version
	upgradeVar.ID = string(clusterVersion.Object.Spec.ClusterID)
	upgradeVar.Name = targetSnoClusterName
	upgradeVar.Operators = installedCSV
	upgradeVar.NodeName = node[0].Object.Name

	return nil
}
