package cluster

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExecCmd executes a command on the provided client on each node matching nodeSelector, retrying on internal errors
// retries times with a 10 second delay between retries, and ignores the stdout.
func ExecCmd(client *clients.Settings, retries uint, nodeSelector, command string) error {
	for retry := range retries {
		err := cluster.ExecCmd(client, nodeSelector, command)
		if isErrorExecuting(err) {
			glog.V(ranparam.LogLevel).Infof("Error during command execution, retry %d (%d max): %w", retry+1, retries, err)

			time.Sleep(10 * time.Second)

			continue
		}

		return err
	}

	return fmt.Errorf("ran out of %d retries executing command %s", retries, command)
}

// ExecCmdWithStdout executes a command on the provided client, retrying on internal errors retries times with a 10
// second delay between retries, and returns the stdout for each node.
func ExecCmdWithStdout(
	client *clients.Settings, retries uint, command string, options ...metav1.ListOptions) (map[string]string, error) {
	for retry := range retries {
		outputs, err := cluster.ExecCmdWithStdout(client, command, options...)
		if isErrorExecuting(err) {
			glog.V(ranparam.LogLevel).Infof("Error during command execution, retry %d (%d max): %w", retry+1, retries, err)

			time.Sleep(10 * time.Second)

			continue
		}

		return outputs, err
	}

	return nil, fmt.Errorf("ran out of %d retries executing command %s", retries, command)
}

// ExecCommandOnSNO executes a command on the provided single node client, retrying on internal errors retries times
// with a 10 second delay between retries, and returns the stdout.
func ExecCommandOnSNO(client *clients.Settings, retries uint, command string) (string, error) {
	outputs, err := ExecCmdWithStdout(client, retries, command)
	if err != nil {
		return "", err
	}

	if len(outputs) != 1 {
		return "", fmt.Errorf("expected results from one node, found %d nodes", len(outputs))
	}

	for _, output := range outputs {
		return output, nil
	}

	return "", fmt.Errorf("found unreachable code in ExecCommandOnSNO")
}

// isErrorExecuting matches errors that contain the message "error executing command in container".
func isErrorExecuting(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "error executing command in container") ||
		strings.Contains(err.Error(), "container not found")
}
