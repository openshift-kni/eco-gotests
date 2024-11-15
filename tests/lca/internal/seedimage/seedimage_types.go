package seedimage

import (
	"github.com/openshift-kni/lifecycle-agent/lca-cli/seedclusterinfo"
	configv1 "github.com/openshift/api/config/v1"
)

// SeedImageContent contains the seed image manifest and proxy info.
type SeedImageContent struct {
	*seedclusterinfo.SeedClusterInfo
	Proxy struct {
		HTTPSProxy string
		HTTPProxy  string
		NOProxy    string
	}
	MirrorConfig *configv1.ImageDigestMirrorSet
}

// ImageInspect contains the fields for unmarshalling podman container image's labels.
type ImageInspect struct {
	Labels map[string]string `json:"Labels"`
}
