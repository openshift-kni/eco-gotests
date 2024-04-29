package cluster

import (
	"errors"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
)

// ExecCommandOnSNO executes a command on a single node cluster and returns the stdout.
func ExecCommandOnSNO(client *clients.Settings, command string) (string, error) {
	outputs, err := cluster.ExecCmdWithStdout(client, command)
	if err != nil {
		return "", err
	}

	if len(outputs) != 1 {
		return "", errors.New("expected results from only one node")
	}

	for _, output := range outputs {
		return output, nil
	}

	// unreachable
	return "", errors.New("found unreachable code in ExecCommandOnSNO")
}
