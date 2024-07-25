package helper

import (
	"fmt"
	"strings"

	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/diskencryption/internal/parsehelper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

const (
	// DiskPrefix linux disk device prefix.
	DiskPrefix = "/dev/"
)

// GetClevisLuksListOutput Run the clevis luks list -d /dev/sdX command and
// returns the output.
func GetClevisLuksListOutput() (string, error) {
	rootDisk, err := getRootDisk()
	if err != nil {
		return "", err
	}

	cmdToExec := fmt.Sprintf("sudo clevis luks list -d %s", rootDisk)

	return cluster.ExecCommandOnSNO(Spoke1APIClient, 3, cmdToExec)
}

// getRootDisk returns the name of the encrypted root disk in the form /dev/sdaX.
func getRootDisk() (string, error) {
	lsblkoutput, err := getAllDriveListOutput()

	if err != nil {
		return "", err
	}

	driveList := parsehelper.GetEncryptedDriveList(lsblkoutput)

	for _, name := range driveList {
		var mounts string
		mounts, err = getLSBLKMounts(DiskPrefix + name)

		if err != nil {
			return "", err
		}

		if parsehelper.IsDiskRoot(mounts) {
			return DiskPrefix + name, nil
		}
	}

	return "", fmt.Errorf("could not find LUKS encrypted root disk")
}

// IsTTYConsole is true if the TTY console is configure on the kernel command line,
// false otherwise.
func IsTTYConsole() (bool, error) {
	cmdToExec := "sudo cat /proc/cmdline"
	output, err := cluster.ExecCommandOnSNO(Spoke1APIClient, 3, cmdToExec)

	if err != nil {
		return false, fmt.Errorf("error getting kernel command line, err: %w", err)
	}

	if strings.Contains(output, "nomodeset") &&
		strings.Contains(output, "console=tty0") &&
		strings.Contains(output, "console=ttyS0,115200n8") {
		return true, nil
	}

	return false, nil
}

// getAllDriveListOutput returns the output of the lsblk -o NAME,FSTYPE -l command.
func getAllDriveListOutput() (string, error) {
	cmdToExec := "lsblk -o NAME,FSTYPE -l"

	return cluster.ExecCommandOnSNO(Spoke1APIClient, 3, cmdToExec)
}

// getLSBLKMounts returns the output of the lsblk -o mountpoints -l /dev/sdaX command on host.
func getLSBLKMounts(diskName string) (string, error) {
	cmdToExec := "lsblk -o mountpoints -l " + diskName

	return cluster.ExecCommandOnSNO(Spoke1APIClient, 3, cmdToExec)
}
