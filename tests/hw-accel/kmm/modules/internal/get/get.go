package get

import (
	"fmt"

	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/nodes"
)

// NumberOfNodesForSelector returns the number or worker nodes.
func NumberOfNodesForSelector(apiClient *clients.Settings, selector map[string]string) (int, error) {
	nodeBuilder := nodes.NewBuilder(apiClient, selector)

	if err := nodeBuilder.Discover(); err != nil {
		fmt.Println("could not discover number of nodes")

		return 0, err
	}

	glog.V(kmmparams.KmmLogLevel).Infof(
		"NumberOfNodesForSelector return %v nodes", len(nodeBuilder.Objects))

	return len(nodeBuilder.Objects), nil
}
