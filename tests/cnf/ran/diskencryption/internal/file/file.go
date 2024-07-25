package file

import (
	"fmt"

	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/cluster"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
)

// TouchFile touches the file passed in parameter.
func TouchFile(path string) error {
	cmdToExec := fmt.Sprintf("sudo touch %s", path)

	return cluster.ExecCmd(Spoke1APIClient, 3, RANConfig.ControlPlaneLabel, cmdToExec)
}

// DeleteFile deletes the file passed in parameter.
func DeleteFile(path string) error {
	cmdToExec := fmt.Sprintf("sudo rm -f %s", path)
	err := cluster.ExecCmd(Spoke1APIClient, 3, RANConfig.ControlPlaneLabel, cmdToExec)

	if err != nil {
		return fmt.Errorf("error deleting file %s, err=%w", path, err)
	}

	return nil
}
