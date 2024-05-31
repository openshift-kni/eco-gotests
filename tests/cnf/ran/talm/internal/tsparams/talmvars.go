package tsparams

import (
	"fmt"

	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/k8sreporter"
	v1 "k8s.io/api/core/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = append(ranparam.Labels, LabelSuite)

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		TestNamespace: "",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}

	// TalmNonExistentClusterMessage is the condition message for when a cluster is non-existent.
	TalmNonExistentClusterMessage = fmt.Sprintf(
		"Unable to select clusters: cluster %s is not a ManagedCluster", NonExistentClusterName)
	// TalmNonExistentPolicyMessage is the condition message for when a policy is non-existent.
	TalmNonExistentPolicyMessage = fmt.Sprintf("Missing managed policies: [%s]", NonExistentPolicyName)

	// SpokeImageListCommand is the command to generate a list of cached images on the spoke cluster.
	SpokeImageListCommand = fmt.Sprintf(`podman images  --noheading --filter "label=name=%s"`, PreCacheExcludedImage)

	// SpokeImageDeleteCommand is the command to delete PreCacheExcludedImage.
	SpokeImageDeleteCommand = fmt.Sprintf(
		`podman images --noheading  --filter "label=name=%s" --format {{.ID}}|xargs podman rmi --force`,
		PreCacheExcludedImage)

	// Spoke1Name is the name of the first spoke cluster.
	Spoke1Name string
	// Spoke2Name is the name of the second spoke cluster.
	Spoke2Name string
	// TalmVersion talm version.
	TalmVersion string
)
