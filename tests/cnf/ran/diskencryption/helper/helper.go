package helper

import (
	"fmt"

	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/parsehelper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
)

const (
	// DiskPrefix linux disk device prefix.
	DiskPrefix = "/dev/"
)

// getAllDriveListOutput returns the output of the lsblk -o NAME,FSTYPE -l command.
func getAllDriveListOutput() (output string, err error) {
	cmdToExec := "lsblk -o NAME,FSTYPE -l"

	return execSNO(cmdToExec)
}

// getLSBLKMounts returns the output of the lsblk -o mountpoints -l /dev/sdaX command on host.
func getLSBLKMounts(diskName string) (mounts string, err error) {
	cmdToExec := "lsblk -o mountpoints -l " + diskName

	return execSNO(cmdToExec)
}

// GetClevisLuksListOutput Run the clevis luks list -d /dev/sdX command and
// returns the output.
func GetClevisLuksListOutput() (output string, err error) {
	rootDisk, err := getRootDisk()
	if err != nil {
		return output, err
	}

	cmdToExec := fmt.Sprintf("sudo clevis luks list -d %s", rootDisk)

	return execSNO(cmdToExec)
}

// getRootDisk returns the name of the encrypted root disk in the form /dev/sdaX.
func getRootDisk() (driveName string, err error) {
	var lsblkoutput string
	lsblkoutput, err = getAllDriveListOutput()

	if err != nil {
		return driveName, err
	}

	driveList := parsehelper.GetEncryptedDriveList(lsblkoutput)

	for _, name := range driveList {
		var mounts string
		mounts, err = getLSBLKMounts(DiskPrefix + name)

		if err != nil {
			return driveName, err
		}

		if parsehelper.IsDiskRoot(mounts) {
			return DiskPrefix + name, nil
		}
	}

	return "", fmt.Errorf("could not find LUKS encrypted root disk")
}

// execSNO executes the cmdToExec in the cluster and returns the output
// of the command only for the first node.
func execSNO(cmdToExec string) (outputFirstNode string, err error) {
	output, err := cluster.ExecCmdWithStdout(raninittools.Spoke1APIClient, cmdToExec)
	if err != nil {
		return outputFirstNode, err
	}

	for _, outputFirstNode = range output {
		break
	}

	return outputFirstNode, nil
}

// SoftRebootSNO executes systemctl reboot on a node.
func SoftRebootSNO() error {
	cmdToExec := "sudo systemctl reboot"

	_, err := execSNO(cmdToExec)

	return err
}
