package seedimage

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/lifecycle-agent/lca-cli/seedclusterinfo"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/nodes"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/cluster"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	seedImageLabel = "com.openshift.lifecycle-agent.seed_cluster_info"
)

// GetContent returns the structured contents of a seed image as SeedImageContent.
//
//nolint:funlen
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

	if seedInfo.HasProxy {
		podmanPullCmd := fmt.Sprintf("%s podman pull %s", connectionString, seedImageLocation)

		_, err = cluster.ExecCmdWithStdout(
			apiClient, podmanPullCmd, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", seedNode),
			})

		if err != nil {
			return nil, err
		}

		mountedFilePathOutput, err := cluster.ExecCmdWithStdout(
			apiClient, fmt.Sprintf("sudo podman image mount %s", seedImageLocation), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", seedNode),
			})

		if err != nil {
			return nil, err
		}

		defer func() {
			_, err := cluster.ExecCmdWithStdout(
				apiClient, fmt.Sprintf("sudo podman image unmount %s", seedImageLocation), metav1.ListOptions{
					FieldSelector: fmt.Sprintf("metadata.name=%s", seedNode),
				})

			if err != nil {
				glog.V(100).Info("Error occurred while unmounting image")
			}
		}()

		mountedFilePath := regexp.MustCompile(`\n`).ReplaceAllString(mountedFilePathOutput[seedNode], "")

		proxyEnvOutput, err := cluster.ExecCmdWithStdout(
			apiClient, fmt.Sprintf("sudo tar xzf %s/etc.tgz -O etc/mco/proxy.env", mountedFilePath), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", seedNode),
			})

		if err != nil {
			return nil, err
		}

		proxyEnv := proxyEnvOutput[seedNode]

		seedInfo.ParseProxyEnv(proxyEnv)
	}

	return seedInfo, nil
}

// ParseProxyEnv reads a proxy.env config and sets SeedImageContent.Proxy values accordingly.
func (s *SeedImageContent) ParseProxyEnv(config string) {
	httpProxyRE := regexp.MustCompile(`HTTP_PROXY=(.+)`)
	httpProxyResult := httpProxyRE.FindString(config)

	if len(httpProxyResult) > 0 {
		httpProxyKeyVal := strings.Split(httpProxyResult, "=")
		if len(httpProxyKeyVal) == 2 {
			s.Proxy.HTTPProxy = httpProxyKeyVal[1]
		}
	}

	httpsProxyRE := regexp.MustCompile(`HTTPS_PROXY=(.+)`)
	httpsProxyResult := httpsProxyRE.FindString(config)

	if len(httpsProxyResult) > 0 {
		httpsKeyVal := strings.Split(httpsProxyResult, "=")
		if len(httpsKeyVal) == 2 {
			s.Proxy.HTTPSProxy = httpsKeyVal[1]
		}
	}

	noProxyRE := regexp.MustCompile(`NO_PROXY=(.*)`)
	noProxyResult := noProxyRE.FindString(config)

	if len(noProxyResult) > 0 {
		noProxyKeyVal := strings.Split(noProxyResult, "=")
		if len(noProxyKeyVal) == 2 {
			s.Proxy.NOProxy = noProxyKeyVal[1]
		}
	}
}
