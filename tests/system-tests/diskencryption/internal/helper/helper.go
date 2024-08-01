package helper

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/diskencryption/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsinittools"
)

const (
	// DiskPrefix linux disk device prefix.
	DiskPrefix = "/dev/"
	// TPM2ReservedSlot TPMv2 reserved slot.
	TPM2ReservedSlot = "31"
	// TPM2ReservedSlotContent TPMv2 reserved slot configuration (to disable PCR protection).
	TPM2ReservedSlotContent = `: tpm2 '{"hash":"sha256","key":"ecc"}'`
)

// GetClevisLuksListOutput Run the clevis luks list -d /dev/sdX command and
// returns the output.
func GetClevisLuksListOutput() (string, error) {
	rootDisk, err := getRootDisk()
	if err != nil {
		return "", err
	}

	cmdToExec := fmt.Sprintf("sudo clevis luks list -d %s", rootDisk)

	return cluster.ExecCommandOnSNOWithRetries(APIClient, tsparams.RetryCount, tsparams.RetryInterval, cmdToExec)
}

// getRootDisk returns the name of the encrypted root disk in the form /dev/sdaX.
func getRootDisk() (string, error) {
	lsblkoutput, err := getAllDriveListOutput()

	if err != nil {
		return "", err
	}

	driveList := GetEncryptedDriveList(lsblkoutput)

	for _, name := range driveList {
		var mounts string
		mounts, err = getLSBLKMounts(DiskPrefix + name)

		if err != nil {
			return "", err
		}

		if IsDiskRoot(mounts) {
			return DiskPrefix + name, nil
		}
	}

	return "", fmt.Errorf("could not find LUKS encrypted root disk")
}

// IsTTYConsole is true if the TTY console is configure on the kernel command line,
// false otherwise.
func IsTTYConsole() (bool, error) {
	cmdToExec := "sudo cat /proc/cmdline"
	output, err := cluster.ExecCommandOnSNOWithRetries(APIClient, tsparams.RetryCount,
		tsparams.RetryInterval, cmdToExec)

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

// GetEncryptedDriveList returns the list of encrypted drives present in the host.
func GetEncryptedDriveList(lsblkoutput string) []string {
	const regex = `(.*?)\scrypto_LUKS`

	// Compile the regular expression
	re := regexp.MustCompile(regex)

	// Find all matches
	matches := re.FindAllStringSubmatch(lsblkoutput, -1)

	var driveList []string

	for _, match := range matches {
		driveList = append(driveList, match[1])
	}

	return driveList
}

// IsDiskRoot returs true if the "diskName" drive is the root drive (e.g. /).
// processes the output of the lsblk -o mountpoints -l /dev/sdaX command.
func IsDiskRoot(lsblkMounts string) bool {
	const regex = `\n/\n`

	// Compile the regular expression
	re := regexp.MustCompile(regex)

	// Find all matches
	matches := re.FindAllStringSubmatch(lsblkMounts, -1)

	return len(matches) > 0
}

var registersPCR = []string{"1", "7"}

// LuksListContainsPCR1And7 checks the output of
// sudo clevis luks list -d /dev/sdX for PCR 1 and 7 configuration.
func LuksListContainsPCR1And7(input string) (found bool) {
	const regex = `[0-9]+:\s+tpm2.*?pcr_ids":"(.*)"`

	// Compile the regular expression
	re := regexp.MustCompile(regex)

	// Find all matches
	matches := re.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		pcrList := strings.Split(match[1], ",")
		if SubSlice(pcrList, registersPCR) {
			return true
		}
	}

	return false
}

// LuksListContainsReservedSlot checks the output of
// sudo clevis luks list -d /dev/sdX for the reserved slot.
func LuksListContainsReservedSlot(input string) bool {
	RefReservedSlot := TPM2ReservedSlot + TPM2ReservedSlotContent

	return RefReservedSlot == input
}

// StringInSlice checks a slice for a given string.
func StringInSlice[T ~string](s []T, str T, contains bool) bool {
	for _, value := range s {
		if !contains {
			if strings.TrimSpace(string(value)) == string(str) {
				return true
			}
		} else {
			if strings.Contains(strings.TrimSpace(string(value)), string(str)) {
				return true
			}
		}
	}

	return false
}

// SubSlice checks if a slice's elements all exist within a slice.
func SubSlice(s, sub []string) bool {
	for _, v := range sub {
		if !StringInSlice(s, v, false) {
			return false
		}
	}

	return true
}

// SwapFirstAndSecondSliceItems swaps the first and second items in a string slice.
func SwapFirstAndSecondSliceItems(slice []string) ([]string, error) {
	if len(slice) < 2 {
		return slice, fmt.Errorf("cannot swap two first items of slice, for slices of length < 2")
	}

	newSlice := slice

	temp := slice[0]
	newSlice[0] = slice[1]
	newSlice[1] = temp

	return newSlice, nil
}

// getAllDriveListOutput returns the output of the lsblk -o NAME,FSTYPE -l command.
func getAllDriveListOutput() (string, error) {
	cmdToExec := "lsblk -o NAME,FSTYPE -l"

	return cluster.ExecCommandOnSNOWithRetries(APIClient, tsparams.RetryCount,
		tsparams.RetryInterval, cmdToExec)
}

// getLSBLKMounts returns the output of the lsblk -o mountpoints -l /dev/sdaX command on host.
func getLSBLKMounts(diskName string) (string, error) {
	cmdToExec := "lsblk -o mountpoints -l " + diskName

	return cluster.ExecCommandOnSNOWithRetries(APIClient, tsparams.RetryCount,
		tsparams.RetryInterval,
		cmdToExec)
}
