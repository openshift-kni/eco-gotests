package reboot

import (
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/cmd"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsinittools"
)

// KernelCrashKdump triggers a kernel crash dump which generates a vmcore dump.
func KernelCrashKdump(nodeName string) error {
	// pull openshift apiserver deployment object to wait for after the node reboot.
	openshiftAPIDeploy, err := deployment.Pull(APIClient, "apiserver", "openshift-apiserver")

	if err != nil {
		return err
	}

	cmdToExec := []string{"chroot", "/rootfs", "/bin/sh", "-c", "rm -rf /var/crash/*"}
	glog.V(90).Infof("Remove any existing crash dumps. Exec cmd %v", cmdToExec)
	_, err = cmd.ExecCmdOnNode(cmdToExec, nodeName)

	if err != nil {
		return err
	}

	cmdToExec = []string{"/bin/sh", "-c", "echo c > /proc/sysrq-trigger"}

	glog.V(90).Infof("Trigerring kernel crash. Exec cmd %v", cmdToExec)
	_, err = cmd.ExecCmdOnNode(cmdToExec, nodeName)

	if err != nil {
		return err
	}

	// wait for the openshift apiserver deployment to be available
	err = openshiftAPIDeploy.WaitUntilCondition("Available", 5*time.Minute)

	if err != nil {
		return err
	}

	return nil
}
