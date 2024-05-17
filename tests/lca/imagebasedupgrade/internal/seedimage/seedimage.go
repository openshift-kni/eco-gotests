package seedimage

import (
	"encoding/json"
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/lifecycle-agent/lca-cli/seedclusterinfo"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	seedImageLabel = "com.openshift.lifecycle-agent.seed_cluster_info"
)

// GetContent returns the structured contents of a seed image as SeedImageContent.
func GetContent(apiClient *clients.Settings, seedImageLocation string) (*SeedImageContent, error) {
	if apiClient == nil {
		return nil, fmt.Errorf("nil apiclient passed to seed image function")
	}

	if seedImageLocation == "" {
		return nil, fmt.Errorf("empty seed image location passed to seed image function")
	}

	ibuNodes, err := nodes.List(apiClient)
	if err != nil {
		return nil, err
	}

	if len(ibuNodes) == 0 {
		return nil, fmt.Errorf("node list was empty")
	}

	seedNode := ibuNodes[0].Object.Name

	targetProxy, err := cluster.GetOCPProxy(apiClient)
	if err != nil {
		return nil, err
	}

	var connectionString string

	switch {
	case len(targetProxy.Object.Spec.HTTPSProxy) != 0:
		connectionString =
			fmt.Sprintf("sudo HTTPS_PROXY=%s", targetProxy.Object.Spec.HTTPSProxy)
	case len(targetProxy.Object.Spec.HTTPProxy) != 0:
		connectionString =
			fmt.Sprintf("sudo HTTP_PROXY=%s", targetProxy.Object.Spec.HTTPProxy)
	default:
		connectionString = "sudo"
	}

	skopeoInspectCmd := fmt.Sprintf("%s skopeo inspect docker://%s", connectionString, seedImageLocation)

	skopeoInspectJSONOutput, err := cluster.ExecCmdWithStdout(
		apiClient, skopeoInspectCmd, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", seedNode),
		})
	if err != nil {
		return nil, err
	}

	skopeoInspectJSON := skopeoInspectJSONOutput[seedNode]

	var imageMeta ImageInspect

	err = json.Unmarshal([]byte(skopeoInspectJSON), &imageMeta)
	if err != nil {
		return nil, err
	}

	if _, ok := imageMeta.Labels[seedImageLabel]; !ok {
		return nil, fmt.Errorf("%s image did not contain expected label: %s", seedImageLocation, seedImageLabel)
	}

	seedInfo := new(SeedImageContent)
	seedInfo.SeedClusterInfo = new(seedclusterinfo.SeedClusterInfo)

	err = json.Unmarshal([]byte(imageMeta.Labels[seedImageLabel]), seedInfo.SeedClusterInfo)
	if err != nil {
		return nil, err
	}

	return seedInfo, nil
}
