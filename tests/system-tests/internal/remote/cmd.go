package remote

import (
	"fmt"
	"time"

	ssh "github.com/povsister/scp"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsinittools"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// ExecCmdOnNode executes a command on a node.
func ExecCmdOnNode(cmdToExec []string, nodeName string) (string, error) {
	listOptions := v1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
		LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": inittools.GeneralConfig.MCOConfigDaemonName}).String(),
	}

	mcPodList, err := pod.List(inittools.APIClient, inittools.GeneralConfig.MCONamespace, listOptions)
	if err != nil {
		return "", err
	}

	glog.V(90).Infof("Exec cmd %v on pod %s", cmdToExec, mcPodList[0].Definition.Name)
	buf, err := mcPodList[0].ExecCommand(cmdToExec)

	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, buf.String())
	}

	return buf.String(), err
}

// ExecuteOnNodeWithPrivilegedDebugPod executes command on the specific node using privileged debug pod.
func ExecuteOnNodeWithPrivilegedDebugPod(apiClient *clients.Settings, cmd []string, hostname string) (string, error) {
	debugPod := pod.NewBuilder(
		apiClient,
		"debug",
		"openshift-machine-config-operator",
		SystemTestsTestConfig.CNFGoTestsClientImage)

	debugPod, err := debugPod.WithPrivilegedFlag().
		WithHostNetwork().
		WithLabel("kubernetes.io/hostname", hostname).
		CreateAndWaitUntilRunning(1 * time.Minute)
	if err != nil {
		return "", err
	}

	buf, err := debugPod.ExecCommand(cmd)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ExecCmdOnHost executes specific cmd on remote host.
func ExecCmdOnHost(remoteHostname, remoteHostUsername, remoteHostPass, cmd string) (string, error) {
	if remoteHostname == "" {
		glog.V(100).Info("The remoteHostname is empty")

		return "", fmt.Errorf("the remoteHostname could not be empty")
	}

	if remoteHostUsername == "" {
		glog.V(100).Info("The remoteHostUsername is empty")

		return "", fmt.Errorf("the remoteHostUsername could not be empty")
	}

	if remoteHostPass == "" {
		glog.V(100).Info("The remoteHostPass is empty")

		return "", fmt.Errorf("the remoteHostPass could not be empty")
	}

	glog.V(100).Info("Build a SSH config from username/password")

	sshConf := ssh.NewSSHConfigFromPassword(remoteHostUsername, remoteHostPass)

	glog.V(100).Infof("Dial SSH to %s", remoteHostname)

	scpClient, err := ssh.NewClient(remoteHostname, sshConf, &ssh.ClientOption{})

	if err != nil {
		return "", err
	}

	ss, _ := scpClient.NewSession()
	defer ss.Close()

	out, err := ss.Output(cmd)

	if err != nil {
		return "", err
	}

	return string(out), nil
}
